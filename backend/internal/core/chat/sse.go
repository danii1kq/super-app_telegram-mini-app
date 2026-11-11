package chat

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

// Hub fans out refresh signals to open SSE streams.
//
// shortcut: signals live in process memory, so one API replica is assumed.
// Switch to PostgreSQL LISTEN/NOTIFY when a second replica appears.
type Hub struct {
	mu      sync.Mutex
	streams map[int64]map[chan string]struct{}
}

func NewHub() *Hub {
	return &Hub{streams: make(map[int64]map[chan string]struct{})}
}

// Publish notifies every subscriber of a conversation.
func (h *Hub) Publish(conversationID int64, event string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for subscriber := range h.streams[conversationID] {
		select {
		case subscriber <- event:
		default:
			// A slow reader misses a signal, not the message: the client
			// refetches from REST anyway.
		}
	}
}

func (h *Hub) subscribe(conversationID int64) (chan string, func()) {
	channel := make(chan string, 8)
	h.mu.Lock()
	if h.streams[conversationID] == nil {
		h.streams[conversationID] = make(map[chan string]struct{})
	}
	h.streams[conversationID][channel] = struct{}{}
	h.mu.Unlock()

	return channel, func() {
		h.mu.Lock()
		delete(h.streams[conversationID], channel)
		if len(h.streams[conversationID]) == 0 {
			delete(h.streams, conversationID)
		}
		h.mu.Unlock()
		close(channel)
	}
}

// stream keeps an SSE connection open for the viewer's conversations.
func (h *Handler) stream(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UserFrom(r.Context()); !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// The client passes the conversations it currently displays; the stream is
	// a hint channel, not an authorisation boundary.
	conversationID, _ := strconv.ParseInt(r.URL.Query().Get("conversation"), 10, 64)
	channel, unsubscribe := h.events.subscribe(conversationID)
	defer unsubscribe()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-channel:
			fmt.Fprintf(w, "event: %s\ndata: {}\n\n", event)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}
