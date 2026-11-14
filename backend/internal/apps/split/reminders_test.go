package split

import (
	"context"
	"testing"
	"time"
)

func TestReminderAgreementScheduleAndPaymentPause(t *testing.T) {
	s, _ := receiptTestService(t)
	ctx := context.Background()
	if _, err := s.pool.Exec(ctx, `INSERT INTO user_settings(user_id,bot_notifications_opt_in)VALUES(1,true),(2,true)`); err != nil {
		t.Fatal(err)
	}
	g, err := s.CreateGroup(ctx, 1, "Loan", []int64{2})
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.CreateDebt(ctx, 1, g.ID, 1, 2, 100000, "", "loan")
	if err != nil {
		t.Fatal(err)
	}
	date := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	a, err := s.ProposeReminder(ctx, 1, d.ID, 0, date, 7)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.RespondReminder(ctx, 1, d.ID, a.Version, "accept")
	requireReceiptStatus(t, err, 403)
	count := func() int {
		var n int
		if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM notification_delivery WHERE kind='split.debt_reminder'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if err = s.ScheduleReminders(ctx, a.DeadlineAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if count() != 0 {
		t.Fatal("unagreed reminder")
	}
	a, err = s.RespondReminder(ctx, 2, d.ID, a.Version, "accept")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.ScheduleReminders(ctx, a.DeadlineAt.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if count() != 0 {
		t.Fatal("early reminder")
	}
	now := a.DeadlineAt.Add(time.Minute)
	if err = s.ScheduleReminders(ctx, now); err != nil {
		t.Fatal(err)
	}
	if err = s.ScheduleReminders(ctx, now); err != nil {
		t.Fatal(err)
	}
	if count() != 1 {
		t.Fatal("duplicate reminder", count())
	}
	now = now.AddDate(0, 0, 90)
	if err = s.ScheduleReminders(ctx, now); err != nil {
		t.Fatal(err)
	}
	if count() != 2 {
		t.Fatal("missed reminders replayed", count())
	}
	a, err = s.Reminder(ctx, 2, d.ID)
	if err != nil || !a.NextRunAt.After(now) {
		t.Fatal(a, err)
	}
	d, err = s.AddPayment(ctx, 2, d.ID, 10000, "sent")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.ScheduleReminders(ctx, a.NextRunAt.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if count() != 2 {
		t.Fatal("reminder during pending payment")
	}
	d, err = s.RespondPayment(ctx, 1, d.ID, d.Payments[0].ID, false)
	if err != nil {
		t.Fatal(err)
	}
	a, err = s.Reminder(ctx, 2, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.pool.Exec(ctx, `UPDATE user_settings SET bot_notifications_opt_in=false WHERE user_id=2`); err != nil {
		t.Fatal(err)
	}
	if err = s.ScheduleReminders(ctx, a.NextRunAt.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if count() != 2 {
		t.Fatal("ignored bot preference")
	}
	if _, err = s.AddPayment(ctx, 1, d.ID, 100000, "received"); err != nil {
		t.Fatal(err)
	}
	a, err = s.Reminder(ctx, 2, d.ID)
	if err != nil || a.Status != "disabled" {
		t.Fatal("paid reminder not stopped", a, err)
	}
}

func TestReminderCalendarDoesNotDriftAcrossDST(t *testing.T) {
	location, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 10, 24, 18, 0, 0, 0, location)
	now := start.Add(25 * time.Hour)
	next := nextReminderAfter(start, now, location, 1).In(location)
	if next.Hour() != 18 || next.Day() != 26 || !next.After(now) {
		t.Fatal(next)
	}
}
