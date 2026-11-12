package day

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

type WeatherCity struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Country   string  `json:"country"`
	Region    string  `json:"region"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type WeatherDay struct {
	Date string  `json:"date"`
	Code int     `json:"code"`
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
}

type weatherCacheEntry struct {
	raw   []byte
	until time.Time
}

type WeatherClient struct {
	http         *http.Client
	geocodingURL string
	forecastURL  string
	apiKey       string
	mu           sync.Mutex
	cache        map[string]weatherCacheEntry
	slots        chan struct{}
}

func NewWeatherClient(apiKey string) *WeatherClient {
	c := &WeatherClient{apiKey: apiKey, geocodingURL: "https://geocoding-api.open-meteo.com/v1/search", forecastURL: "https://api.open-meteo.com/v1/forecast", cache: make(map[string]weatherCacheEntry), slots: make(chan struct{}, 4), http: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	if apiKey != "" {
		c.geocodingURL = "https://customer-geocoding-api.open-meteo.com/v1/search"
		c.forecastURL = "https://customer-api.open-meteo.com/v1/forecast"
	}
	return c
}

func validCoordinates(latitude, longitude float64) bool {
	return !math.IsNaN(latitude) && !math.IsNaN(longitude) && !math.IsInf(latitude, 0) && !math.IsInf(longitude, 0) && latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180
}

func weatherError() error {
	return httpapi.Conflict("weather_unavailable", "Погода сейчас недоступна, попробуйте позже")
}

func (c *WeatherClient) get(ctx context.Context, base string, query url.Values, target any) error {
	if c.apiKey != "" {
		query.Set("apikey", c.apiKey)
	}
	endpoint := base + "?" + query.Encode()
	c.mu.Lock()
	cached, ok := c.cache[endpoint]
	c.mu.Unlock()
	if ok && time.Now().Before(cached.until) {
		if json.Unmarshal(cached.raw, target) == nil {
			return nil
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	case <-ctx.Done():
		return weatherError()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return weatherError()
	}
	response, err := c.http.Do(req)
	if err != nil {
		return weatherError()
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return weatherError()
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	if err != nil || len(raw) > 64<<10 || json.Unmarshal(raw, target) != nil {
		return weatherError()
	}
	c.mu.Lock()
	// The bounded cache is disposable; no user location history is persisted.
	if len(c.cache) >= 128 {
		clear(c.cache)
	}
	c.cache[endpoint] = weatherCacheEntry{raw: raw, until: time.Now().Add(15 * time.Minute)}
	c.mu.Unlock()
	return nil
}

func (c *WeatherClient) Cities(ctx context.Context, query, language string) ([]WeatherCity, error) {
	query = strings.TrimSpace(query)
	if len([]rune(query)) < 2 || len([]rune(query)) > 80 {
		return nil, httpapi.BadRequest("invalid_city", "Введите название города от 2 до 80 символов")
	}
	if language != "ru" && language != "kk" && language != "en" {
		language = "ru"
	}
	var result struct {
		Results []struct {
			ID        int64   `json:"id"`
			Name      string  `json:"name"`
			Country   string  `json:"country"`
			Region    string  `json:"admin1"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"results"`
	}
	if err := c.get(ctx, c.geocodingURL, url.Values{"name": {query}, "count": {"5"}, "language": {language}, "format": {"json"}}, &result); err != nil {
		return nil, err
	}
	cities := make([]WeatherCity, 0, 5)
	for _, city := range result.Results {
		if city.ID > 0 && city.Name != "" && validCoordinates(city.Latitude, city.Longitude) {
			cities = append(cities, WeatherCity{ID: city.ID, Name: city.Name, Country: city.Country, Region: city.Region, Latitude: city.Latitude, Longitude: city.Longitude})
			if len(cities) == 5 {
				break
			}
		}
	}
	return cities, nil
}

func (c *WeatherClient) Forecast(ctx context.Context, latitude, longitude float64, zone string) ([]WeatherDay, error) {
	if !validCoordinates(latitude, longitude) || len(zone) > 80 {
		return nil, httpapi.BadRequest("invalid_location", "Выберите город и корректный часовой пояс")
	}
	if _, err := time.LoadLocation(zone); err != nil {
		return nil, httpapi.BadRequest("invalid_timezone", "Некорректный часовой пояс")
	}
	var result struct {
		Daily struct {
			Time []string   `json:"time"`
			Code []*int     `json:"weather_code"`
			Min  []*float64 `json:"temperature_2m_min"`
			Max  []*float64 `json:"temperature_2m_max"`
		} `json:"daily"`
	}
	query := url.Values{"latitude": {strconv.FormatFloat(math.Round(latitude*100)/100, 'f', 2, 64)}, "longitude": {strconv.FormatFloat(math.Round(longitude*100)/100, 'f', 2, 64)}, "timezone": {zone}, "daily": {"weather_code,temperature_2m_max,temperature_2m_min"}, "forecast_days": {"16"}}
	if err := c.get(ctx, c.forecastURL, query, &result); err != nil {
		return nil, err
	}
	daily := result.Daily
	if len(daily.Time) == 0 || len(daily.Time) > 16 || len(daily.Code) != len(daily.Time) || len(daily.Min) != len(daily.Time) || len(daily.Max) != len(daily.Time) {
		return nil, weatherError()
	}
	days := make([]WeatherDay, 0, len(daily.Time))
	seen := make(map[string]bool)
	for index, date := range daily.Time {
		if _, err := time.Parse("2006-01-02", date); err != nil || seen[date] {
			return nil, weatherError()
		}
		seen[date] = true
		if daily.Code[index] == nil || daily.Min[index] == nil || daily.Max[index] == nil {
			continue
		}
		if *daily.Code[index] < 0 || *daily.Code[index] > 99 || *daily.Min[index] > *daily.Max[index] {
			return nil, weatherError()
		}
		days = append(days, WeatherDay{Date: date, Code: *daily.Code[index], Min: *daily.Min[index], Max: *daily.Max[index]})
	}
	return days, nil
}

func (h *Handler) registerWeather(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/day/weather/cities", h.auth.Middleware(http.HandlerFunc(h.weatherCities)))
	mux.Handle("GET /api/v1/day/weather", h.auth.Middleware(http.HandlerFunc(h.weatherForecast)))
}

func (h *Handler) weatherCities(w http.ResponseWriter, r *http.Request) {
	cities, err := h.weather.Cities(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("language"))
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"cities": cities})
}

func (h *Handler) weatherForecast(w http.ResponseWriter, r *http.Request) {
	latitude, latErr := strconv.ParseFloat(r.URL.Query().Get("latitude"), 64)
	longitude, lonErr := strconv.ParseFloat(r.URL.Query().Get("longitude"), 64)
	if latErr != nil || lonErr != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_location", "Выберите город"))
		return
	}
	days, err := h.weather.Forecast(r.Context(), latitude, longitude, r.URL.Query().Get("timezone"))
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"days": days})
}
