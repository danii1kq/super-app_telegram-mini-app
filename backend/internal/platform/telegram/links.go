package telegram

import (
	"net/url"
	"regexp"
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
func DeepLink(base, payload string) string {
	if base == "" || !startPattern.MatchString(payload) {
		return ""
	}
	return base + "?startapp=" + url.QueryEscape(payload)
}
