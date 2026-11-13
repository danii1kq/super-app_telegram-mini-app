package day

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

type eventRecord struct {
	ID         int64     `json:"id"`
	OwnerID    int64     `json:"owner_id"`
	GroupID    *int64    `json:"group_id"`
	Title      string    `json:"title"`
	Location   string    `json:"location"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
	AllDay     bool      `json:"all_day"`
	Visibility string    `json:"visibility"`
	Source     string    `json:"source"`
	RRule      string    `json:"rrule"`
	Version    int       `json:"version"`
}

func (record eventRecord) event(viewerID int64) Event {
	return Event{ID: strconv.FormatInt(record.ID, 10), Title: record.Title, Location: record.Location, StartsAt: record.StartsAt, EndsAt: record.EndsAt, AllDay: record.AllDay, Version: record.Version, Recurring: record.RRule != "", Editable: record.OwnerID == viewerID && record.GroupID == nil && record.RRule == "" && record.Visibility == "private" && (record.Source == "manual" || record.Source == "ai_import")}
}

func validRecord(record eventRecord, eventID int64) bool {
	return record.ID == eventID && record.OwnerID > 0 && record.Version > 0 && !record.StartsAt.IsZero() && record.EndsAt.After(record.StartsAt)
}

func (c *Client) eventRecord(ctx context.Context, initData string, eventID int64) (eventRecord, daySession, error) {
	session, err := c.authenticate(ctx, initData)
	if err != nil {
		return eventRecord{}, daySession{}, err
	}
	var record eventRecord
	if err := c.do(ctx, http.MethodGet, "/api/v1/events/"+strconv.FormatInt(eventID, 10), session.Token, nil, &record, 0); err != nil {
		c.forgetOnAuthError(initData, err)
		return eventRecord{}, daySession{}, err
	}
	if !validRecord(record, eventID) {
		return eventRecord{}, daySession{}, httpapi.Conflict("day_error", "Не удалось прочитать событие")
	}
	return record, session, nil
}

func (c *Client) GetEvent(ctx context.Context, initData string, eventID int64) (Event, error) {
	record, session, err := c.eventRecord(ctx, initData, eventID)
	if err != nil {
		return Event{}, err
	}
	return record.event(session.User.ID), nil
}

func (c *Client) UpdateEvent(ctx context.Context, initData string, eventID int64, version int, input CreateEventInput) (Event, error) {
	record, session, err := c.eventRecord(ctx, initData, eventID)
	if err != nil {
		return Event{}, err
	}
	if !record.event(session.User.ID).Editable {
		return Event{}, httpapi.Forbidden("Можно изменять только свои личные события без повторений")
	}
	if record.Version != version {
		return Event{}, httpapi.Conflict("day_event_changed", "Событие уже изменилось, обновите календарь")
	}
	location := input.Location
	if location == "" {
		location = " "
	} // Upstream treats an empty PATCH string as absent.
	body := map[string]any{"title": input.Title, "starts_at": input.StartsAt, "ends_at": input.EndsAt, "all_day": input.AllDay, "location": location}
	var updated eventRecord
	if err := c.do(ctx, http.MethodPatch, "/api/v1/events/"+strconv.FormatInt(eventID, 10), session.Token, body, &updated, version); err != nil {
		c.forgetOnAuthError(initData, err)
		return Event{}, err
	}
	if !validRecord(updated, eventID) || updated.Version <= version {
		return Event{}, httpapi.Conflict("day_error", "Не удалось подтвердить изменение, обновите календарь перед повтором")
	}
	return updated.event(session.User.ID), nil
}

func (s *Service) Event(ctx context.Context, userID int64, initData string, eventID int64) (Event, error) {
	linked, err := s.Linked(ctx, userID)
	if err != nil {
		return Event{}, err
	}
	if !linked {
		return Event{}, httpapi.Conflict("not_linked", "Подключите свой календарь SruDay")
	}
	return s.client.GetEvent(ctx, initData, eventID)
}

func (s *Service) Update(ctx context.Context, userID int64, initData string, eventID int64, version int, input CreateEventInput) (Event, error) {
	if eventID <= 0 || version <= 0 {
		return Event{}, httpapi.BadRequest("invalid_version", "Обновите событие перед сохранением")
	}
	if err := validateEvent(&input); err != nil {
		return Event{}, err
	}
	linked, err := s.Linked(ctx, userID)
	if err != nil {
		return Event{}, err
	}
	if !linked {
		return Event{}, httpapi.Conflict("not_linked", "Подключите свой календарь SruDay")
	}
	return s.client.UpdateEvent(ctx, initData, eventID, version, input)
}

func eventID(r *http.Request) (int64, error) {
	raw := r.PathValue("id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 || strconv.FormatInt(id, 10) != raw {
		return 0, httpapi.BadRequest("invalid_id", "Некорректное событие")
	}
	return id, nil
}

func (h *Handler) event(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	id, err := eventID(r)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	initData, err := h.auth.InitDataFor(r.Context(), user.ID, auth.BearerToken(r))
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	event, err := h.service.Event(r.Context(), user.ID, initData, id)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, event)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	id, err := eventID(r)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	var body struct {
		Title    string `json:"title"`
		StartsAt string `json:"starts_at"`
		EndsAt   string `json:"ends_at"`
		Location string `json:"location"`
		AllDay   bool   `json:"all_day"`
		Version  int    `json:"version"`
	}
	if err := httpapi.DecodeJSON(w, r, &body, 8<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	initData, err := h.auth.InitDataFor(r.Context(), user.ID, auth.BearerToken(r))
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	event, err := h.service.Update(r.Context(), user.ID, initData, id, body.Version, CreateEventInput{Title: body.Title, StartsAt: body.StartsAt, EndsAt: body.EndsAt, Location: body.Location, AllDay: body.AllDay})
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, event)
}
