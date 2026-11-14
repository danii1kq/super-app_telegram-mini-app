package split

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/jackc/pgx/v5"
)

type ReminderAgreement struct {
	DebtID       string    `json:"debt_id"`
	ProposerID   string    `json:"proposer_id"`
	Version      int       `json:"version"`
	DeadlineAt   time.Time `json:"deadline_at"`
	Timezone     string    `json:"timezone"`
	IntervalDays int       `json:"interval_days"`
	Status       string    `json:"status"`
	NextRunAt    time.Time `json:"next_run_at"`
}

func readReminder(ctx context.Context, q receiptQuerier, debtID int64) (*ReminderAgreement, error) {
	a := &ReminderAgreement{DebtID: strconv.FormatInt(debtID, 10)}
	var proposer int64
	err := q.QueryRow(ctx, `SELECT proposer_id,version,deadline_at,timezone,interval_days,status,next_run_at FROM split_reminder_agreements WHERE debt_id=$1`, debtID).Scan(&proposer, &a.Version, &a.DeadlineAt, &a.Timezone, &a.IntervalDays, &a.Status, &a.NextRunAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	a.ProposerID = strconv.FormatInt(proposer, 10)
	return a, err
}
func (s *Service) Reminder(ctx context.Context, userID, debtID int64) (*ReminderAgreement, error) {
	d, err := s.debtsByID(ctx, userID, debtID)
	if err != nil {
		return nil, err
	}
	if userID != d.CreditorID && userID != d.DebtorID {
		return nil, httpapi.NotFound("Долг не найден")
	}
	return readReminder(ctx, s.pool, debtID)
}

func (s *Service) ProposeReminder(ctx context.Context, userID, debtID int64, expected int, date string, days int) (*ReminderAgreement, error) {
	if days != 1 && days != 3 && days != 7 {
		return nil, httpapi.BadRequest("invalid_frequency", "Выберите 1, 3 или 7 дней")
	}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		d, err := s.lockedDebt(ctx, tx, userID, debtID)
		if err != nil {
			return err
		}
		_, paid, err := s.paymentsOfTx(ctx, tx, debtID)
		if err != nil {
			return err
		}
		if paid >= d.AmountMinor {
			return httpapi.Conflict("debt_settled", "Долг уже закрыт")
		}
		old, err := readReminder(ctx, tx, debtID)
		if err != nil {
			return err
		}
		version := 0
		if old != nil {
			version = old.Version
		}
		if version != expected {
			return httpapi.Conflict("version_conflict", "Условия изменились. Обновите их")
		}
		var zone string
		if err = tx.QueryRow(ctx, `SELECT timezone FROM user_settings WHERE user_id=$1`, d.DebtorID).Scan(&zone); err != nil {
			return err
		}
		location, err := time.LoadLocation(zone)
		if err != nil {
			return httpapi.BadRequest("invalid_timezone", "Укажите корректный часовой пояс в профиле")
		}
		day, err := time.ParseInLocation("2006-01-02", date, location)
		if err != nil || date < time.Now().In(location).Format("2006-01-02") {
			return httpapi.BadRequest("invalid_deadline", "Выберите сегодняшний или будущий день")
		}
		deadline := time.Date(day.Year(), day.Month(), day.Day(), 18, 0, 0, 0, location)
		_, err = tx.Exec(ctx, `INSERT INTO split_reminder_agreements(debt_id,proposer_id,version,deadline_at,timezone,interval_days,next_run_at)VALUES($1,$2,$3,$4,$5,$6,$4) ON CONFLICT(debt_id)DO UPDATE SET proposer_id=$2,version=$3,deadline_at=$4,timezone=$5,interval_days=$6,next_run_at=$4,status='proposed',updated_at=now()`, debtID, userID, version+1, deadline.UTC(), zone, days)
		if err != nil {
			return err
		}
		return cancelReminders(ctx, tx, debtID)
	})
	if err != nil {
		return nil, err
	}
	return readReminder(ctx, s.pool, debtID)
}
func cancelReminders(ctx context.Context, tx pgx.Tx, debtID int64) error {
	_, err := tx.Exec(ctx, `UPDATE notification_delivery SET state='skipped',last_error='reminder agreement changed' WHERE kind='split.debt_reminder' AND subject=$1 AND state='pending'`, "debt:"+strconv.FormatInt(debtID, 10))
	return err
}

func (s *Service) RespondReminder(ctx context.Context, userID, debtID int64, version int, action string) (*ReminderAgreement, error) {
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		if _, err := s.lockedDebt(ctx, tx, userID, debtID); err != nil {
			return err
		}
		a, err := readReminder(ctx, tx, debtID)
		if err != nil {
			return err
		}
		if a == nil {
			return httpapi.NotFound("Условия не найдены")
		}
		if a.Version != version {
			return httpapi.Conflict("version_conflict", "Условия изменились")
		}
		status := "disabled"
		switch action {
		case "accept":
			if a.ProposerID == strconv.FormatInt(userID, 10) {
				return httpapi.Forbidden("Условия подтверждает вторая сторона")
			}
			if a.Status == "active" {
				return nil
			}
			if a.Status != "proposed" {
				return httpapi.Conflict("agreement_disabled", "Предложите условия заново")
			}
			status = "active"
		case "disable":
		default:
			return httpapi.BadRequest("invalid_action", "Некорректное действие")
		}
		if _, err = tx.Exec(ctx, `UPDATE split_reminder_agreements SET status=$2,updated_at=now() WHERE debt_id=$1`, debtID, status); err != nil {
			return err
		}
		return cancelReminders(ctx, tx, debtID)
	})
	if err != nil {
		return nil, err
	}
	return readReminder(ctx, s.pool, debtID)
}

func nextReminderAfter(scheduled, now time.Time, location *time.Location, days int) time.Time {
	next := scheduled.In(location)
	for !next.After(now) {
		next = next.AddDate(0, 0, days)
	}
	return next.UTC()
}

func (s *Service) ScheduleReminders(ctx context.Context, now time.Time) error {
	rows, err := s.pool.Query(ctx, `SELECT debt_id,proposer_id FROM split_reminder_agreements WHERE status='active' AND next_run_at<=$1 ORDER BY next_run_at,debt_id LIMIT 20`, now)
	if err != nil {
		return err
	}
	type due struct{ id, user int64 }
	dueItems := []due{}
	for rows.Next() {
		var item due
		if err = rows.Scan(&item.id, &item.user); err != nil {
			rows.Close()
			return err
		}
		dueItems = append(dueItems, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range dueItems {
		if err = s.scheduleReminder(ctx, item.user, item.id, now); err != nil {
			return err
		}
	}
	return nil
}
func (s *Service) scheduleReminder(ctx context.Context, userID, debtID int64, now time.Time) error {
	return s.pool.InTx(ctx, func(tx pgx.Tx) error {
		d, err := s.lockedDebt(ctx, tx, userID, debtID)
		if err != nil {
			return err
		}
		a, err := readReminder(ctx, tx, debtID)
		if err != nil {
			return err
		}
		if a == nil || a.Status != "active" || a.NextRunAt.After(now) {
			return nil
		}
		payments, paid, err := s.paymentsOfTx(ctx, tx, debtID)
		if err != nil {
			return err
		}
		if paid >= d.AmountMinor {
			if _, err = tx.Exec(ctx, `UPDATE split_reminder_agreements SET status='disabled' WHERE debt_id=$1`, debtID); err != nil {
				return err
			}
			return cancelReminders(ctx, tx, debtID)
		}
		location, err := time.LoadLocation(a.Timezone)
		if err != nil {
			return err
		}
		next := nextReminderAfter(a.NextRunAt, now, location, a.IntervalDays)
		if _, err = tx.Exec(ctx, `UPDATE split_reminder_agreements SET next_run_at=$2 WHERE debt_id=$1`, debtID, next); err != nil {
			return err
		}
		for _, p := range payments {
			if p.Status == "pending" {
				return nil
			}
		}
		var allowed, enabled, amounts bool
		var lang string
		if err = tx.QueryRow(ctx, `SELECT bot_notifications_opt_in,split_debt_notifications,split_debt_show_amounts,lang FROM user_settings WHERE user_id=$1`, d.DebtorID).Scan(&allowed, &enabled, &amounts, &lang); err != nil {
			return err
		}
		if !allowed || !enabled {
			return nil
		}
		d.Outstanding = d.AmountMinor - paid
		title := "SruSplit: напоминание по согласованному долгу. Откройте приложение, чтобы посмотреть остаток."
		button := "Открыть долг"
		if lang == "kk" {
			title = "SruSplit: келісілген қарыз туралы еске салу. Қалдықты қолданбадан қараңыз."
			button = "Қарызды ашу"
		} else if lang == "en" {
			title = "SruSplit: agreed debt reminder. Open the app to see the remaining amount."
			button = "Open debt"
		}
		if amounts {
			switch lang {
			case "kk":
				title = "SruSplit: келісілген қарызды қайтаруды ұмытпаңыз. Қалдық: " + formatMoney(d.Outstanding) + "."
			case "en":
				title = "SruSplit: agreed repayment reminder. Remaining: " + formatMoney(d.Outstanding) + "."
			default:
				title = "SruSplit: по договорённости напоминаем о возврате. Остаток: " + formatMoney(d.Outstanding) + "."
			}
		}
		notice := DebtNotice{Text: title, ButtonURL: "/apps/split/" + strconv.FormatInt(d.GroupID, 10) + "/debts", ButtonText: button, ReminderVersion: a.Version}
		key := "reminder:" + strconv.FormatInt(debtID, 10) + ":" + strconv.Itoa(a.Version) + ":" + a.NextRunAt.UTC().Format(time.RFC3339)
		return s.enqueueNotice(ctx, tx, d.DebtorID, "split.debt_reminder", key, "debt:"+strconv.FormatInt(debtID, 10), d.Version, notice)
	})
}
func (s *Service) RunReminders(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if err := s.ScheduleReminders(ctx, now); err != nil {
				slog.Error("split reminder schedule failed")
			}
		}
	}
}
