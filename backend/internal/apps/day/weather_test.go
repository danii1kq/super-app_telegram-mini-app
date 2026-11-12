package day

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestWeatherContractsCacheAndLocalDates(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("Authorization") != "" {
			t.Error("must not send SruNal authorization to weather provider")
		}
		query := r.URL.Query()
		if r.URL.Path == "/cities" {
			if query.Get("name") != "Алматы" || query.Get("count") != "5" || query.Get("language") != "ru" {
				t.Error("wrong geocoding contract")
			}
			w.Write([]byte(`{"results":[{"id":1526384,"name":"Алматы","admin1":"Алматы","country":"Казахстан","latitude":43.2567,"longitude":76.9286}]}`))
			return
		}
		if query.Get("daily") != "weather_code,temperature_2m_max,temperature_2m_min" || query.Get("timezone") != "Asia/Qyzylorda" || query.Get("forecast_days") != "16" || query.Get("latitude") != "43.26" {
			t.Error("wrong daily forecast contract")
		}
		w.Write([]byte(`{"daily":{"time":["2026-11-13","2026-11-14"],"weather_code":[3,null],"temperature_2m_min":[-2,null],"temperature_2m_max":[8,null]}}`))
	}))
	defer server.Close()
	client := NewWeatherClient("")
	client.geocodingURL = server.URL + "/cities"
	client.forecastURL = server.URL + "/forecast"
	cities, err := client.Cities(context.Background(), " Алматы ", "bad")
	if err != nil || len(cities) != 1 || cities[0].ID != 1526384 {
		t.Fatalf("cities: %#v %v", cities, err)
	}
	for range 2 {
		days, err := client.Forecast(context.Background(), cities[0].Latitude, cities[0].Longitude, "Asia/Qyzylorda")
		if err != nil || len(days) != 1 || days[0].Date != "2026-11-13" || days[0].Min != -2 || days[0].Max != 8 {
			t.Fatalf("days: %#v %v", days, err)
		}
	}
	if requests.Load() != 2 {
		t.Fatal("repeat forecast should use bounded cache")
	}
}

func TestWeatherValidationAndFailureIsolation(t *testing.T) {
	client := NewWeatherClient("private-weather-key")
	if !strings.Contains(client.forecastURL, "customer-api.open-meteo.com") || !strings.Contains(client.geocodingURL, "customer-geocoding-api.open-meteo.com") {
		t.Fatal("commercial key must use customer endpoints")
	}
	for _, query := range []string{" ", "A", strings.Repeat("a", 81)} {
		if _, err := client.Cities(context.Background(), query, "ru"); err == nil {
			t.Fatal("invalid city accepted")
		}
	}
	for _, coords := range [][2]float64{{math.NaN(), 0}, {0, math.Inf(1)}, {91, 0}, {0, -181}} {
		if _, err := client.Forecast(context.Background(), coords[0], coords[1], "UTC"); err == nil {
			t.Fatal("invalid location accepted")
		}
	}
	if _, err := client.Forecast(context.Background(), 1, 2, "Not/AZone"); err == nil {
		t.Fatal("invalid timezone accepted")
	}
	for _, body := range []string{`{"daily":{"time":["2026-11-13"],"weather_code":[],"temperature_2m_min":[0],"temperature_2m_max":[1]}}`, `not json`, `{"daily":{"time":["bad"],"weather_code":[0],"temperature_2m_min":[0],"temperature_2m_max":[1]}}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
		client.forecastURL = server.URL
		clear(client.cache)
		if _, err := client.Forecast(context.Background(), 1, 2, "UTC"); err == nil || strings.Contains(err.Error(), "private-weather-key") {
			t.Fatal("broken provider accepted or secret leaked")
		}
		server.Close()
	}
	targetHit := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetHit = true }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	client.forecastURL = redirect.URL
	clear(client.cache)
	if _, err := client.Forecast(context.Background(), 1, 2, "UTC"); err == nil || targetHit {
		t.Fatal("redirect must not forward weather credentials")
	}
}
