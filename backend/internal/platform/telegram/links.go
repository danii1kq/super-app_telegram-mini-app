package telegram

import (
	"net/url"
	"regexp"
	"strings"
)

var botNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{4,31}$`)
var appNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,64}$`)
var startPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,512}$`)

func MiniAppURL(botName, shortName string) string {
	if !botNamePattern.MatchString(botName) {
		return ""
	}
	result := "https://t.me/" + botName
	if shortName != "" {
		if !appNamePattern.MatchString(shortName) {
			return ""
		}
		result += "/" + shortName
	}
	return result
}

func NotificationLink(base, route string) string {
	if base == "" {
		return ""
	}
	if strings.HasPrefix(route, base+"?startapp=") {
		payload := strings.TrimPrefix(route, base+"?startapp=")
		return DeepLink(base, payload)
	}
	parts := strings.Split(strings.Trim(route, "/"), "/")
	if len(parts) == 4 && parts[0] == "apps" && parts[1] == "split" && parts[3] == "debts" && regexp.MustCompile(`^\d+$`).MatchString(parts[2]) {
		return DeepLink(base, "split_debts_"+parts[2])
	}
	if len(parts) == 5 && parts[0] == "apps" && parts[1] == "split" && parts[3] == "receipts" && regexp.MustCompile(`^\d+$`).MatchString(parts[4]) {
		return DeepLink(base, "split_receipt_"+parts[4])
	}
	return ""
}
func DeepLink(base, payload string) string {
	if base == "" || !startPattern.MatchString(payload) {
		return ""
	}
	return base + "?startapp=" + url.QueryEscape(payload)
}
