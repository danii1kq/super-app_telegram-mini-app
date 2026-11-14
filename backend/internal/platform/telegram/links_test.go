package telegram

import "testing"

func TestNotificationLinksAreTelegramMiniAppLinks(t *testing.T) {
	base := MiniAppURL("srunal_test_bot", "app")
	if NotificationLink(base, "/apps/split/42/debts") != "https://t.me/srunal_test_bot/app?startapp=split_debts_42" {
		t.Fatal("invalid debt link")
	}
	if NotificationLink(base, "/apps/split/42/receipts/17") != "https://t.me/srunal_test_bot/app?startapp=split_receipt_17" {
		t.Fatal("invalid receipt link")
	}
	for _, value := range []string{"https://evil.test", "javascript:alert(1)", "/apps/split/../debts"} {
		if NotificationLink(base, value) != "" {
			t.Fatal("unsafe link", value)
		}
	}
	if MiniAppURL("https://evil.test", "") != "" || DeepLink("", "payload") != "" {
		t.Fatal("invalid base accepted")
	}
}
