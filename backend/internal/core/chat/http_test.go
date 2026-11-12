package chat

import (
	"encoding/json"
	"net/url"
	"testing"
)

func TestHistoryCursorValidation(t *testing.T) {
	for _, raw := range []string{"after=-1", "after=abc", "before=0", "before=", "after=1&before=2", "limit=0", "limit=101", "limit=NaN", "after=9223372036854775808"} {
		values, _ := url.ParseQuery(raw)
		if _, err := parseHistoryQuery(values); err == nil {
			t.Errorf("accepted invalid history query %q", raw)
		}
	}
	latest, err := parseHistoryQuery(url.Values{})
	if err != nil || latest.After != nil || latest.Before != nil || latest.Limit != 50 {
		t.Fatalf("latest defaults: %+v, %v", latest, err)
	}
	forward, err := parseHistoryQuery(url.Values{"after": {"0"}, "limit": {"100"}})
	if err != nil || forward.After == nil || *forward.After != 0 || forward.Limit != 100 {
		t.Fatalf("explicit zero cursor must retain forward history: %+v, %v", forward, err)
	}
	older, err := parseHistoryQuery(url.Values{"before": {"9007199254740993"}})
	if err != nil || older.Before == nil || *older.Before != 9007199254740993 {
		t.Fatalf("lost 64-bit history cursor: %+v, %v", older, err)
	}
}

func TestReadRequestKeepsIntegerPrecision(t *testing.T) {
	for _, raw := range []string{`{"seq":"9007199254740993"}`, `{"seq":9007199254740993}`} {
		var request readRequest
		if err := json.Unmarshal([]byte(raw), &request); err != nil {
			t.Fatal(err)
		}
		seq, err := request.Seq.Int64()
		if err != nil || seq != 9007199254740993 {
			t.Fatalf("read sequence lost precision: %d, %v", seq, err)
		}
	}
}
