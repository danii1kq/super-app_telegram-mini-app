// Package httpapi holds the shared HTTP plumbing: JSON responses, error
// mapping and middleware. It knows nothing about notes, orders or expenses.
package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// Error is a client-visible failure with a stable code.
type Error struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
}

func (e *Error) Error() string { return e.Message }

func BadRequest(code, message string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: code, Message: message}
}

func Unauthorized(message string) *Error {
	return &Error{Status: http.StatusUnauthorized, Code: "unauthorized", Message: message}
}

func Forbidden(message string) *Error {
	return &Error{Status: http.StatusForbidden, Code: "forbidden", Message: message}
}

func NotFound(message string) *Error {
	return &Error{Status: http.StatusNotFound, Code: "not_found", Message: message}
}

func Conflict(code, message string) *Error {
	return &Error{Status: http.StatusConflict, Code: code, Message: message}
}

func Validation(fields map[string]string, message string) *Error {
	return &Error{Status: http.StatusUnprocessableEntity, Code: "validation_failed", Message: message, Fields: fields}
}

func Internal(err error) *Error {
	slog.Error("internal error", "err", err)
	return &Error{Status: http.StatusInternalServerError, Code: "internal", Message: "Внутренняя ошибка сервера"}
}

// WriteJSON writes a JSON body with the given status.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("encode response", "err", err)
	}
}

// WriteError maps an error to the JSON error envelope.
func WriteError(w http.ResponseWriter, err error) {
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		apiErr = Internal(err)
	}
	body := map[string]any{
		"error": map[string]any{
			"code":    apiErr.Code,
			"message": apiErr.Message,
		},
	}
	if len(apiErr.Fields) > 0 {
		body["error"].(map[string]any)["fields"] = apiErr.Fields
	}
	WriteJSON(w, apiErr.Status, body)
}

// DecodeJSON reads a bounded JSON body into dst.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return BadRequest("invalid_json", "Некорректное тело запроса")
	}
	return nil
}
