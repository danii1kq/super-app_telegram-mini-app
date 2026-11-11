// Package app wires the services together and builds the HTTP handler.
//
// It is the only place that knows every module; the modules never import each
// other directly.
package app

import (
	"context"
	"net/http"

	"github.com/danii1kq/srunal/internal/platform/config"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/postgres"
)

// Deps carries what every module needs.
type Deps struct {
	Config config.Config
	Pool   *postgres.Pool
}

// NewAPI builds the root handler.
func NewAPI(ctx context.Context, cfg config.Config, pool *postgres.Pool) (http.Handler, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			httpapi.WriteError(w, httpapi.Internal(err))
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Unknown API routes answer JSON, never the SPA shell.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		httpapi.WriteError(w, httpapi.NotFound("Маршрут не найден"))
	})

	_ = ctx
	return mux, nil
}
