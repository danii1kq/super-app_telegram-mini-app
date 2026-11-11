// Package app wires the services together and builds the HTTP handler.
//
// It is the only place that knows every module; modules never import each
// other directly.
package app

import (
	"context"
	"net/http"

	"github.com/danii1kq/srunal/internal/apps/day"
	"github.com/danii1kq/srunal/internal/apps/doc"
	"github.com/danii1kq/srunal/internal/apps/notes"
	"github.com/danii1kq/srunal/internal/apps/pulse"
	"github.com/danii1kq/srunal/internal/apps/split"
	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/core/chat"
	"github.com/danii1kq/srunal/internal/core/social"
	"github.com/danii1kq/srunal/internal/core/user"
	"github.com/danii1kq/srunal/internal/integration"
	"github.com/danii1kq/srunal/internal/platform/config"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/danii1kq/srunal/internal/platform/secure"
)

// Deps carries what every module needs.
type Deps struct {
	Config  config.Config
	Pool    *postgres.Pool
	Content *secure.Cipher
	Index   *secure.LookupIndex
}

// NewAPI builds the root handler.
func NewAPI(ctx context.Context, cfg config.Config, pool *postgres.Pool) (http.Handler, error) {
	content, err := secure.NewCipher(cfg.ContentKey)
	if err != nil {
		return nil, err
	}
	index, err := secure.NewLookupIndex(cfg.HMACIndexKey)
	if err != nil {
		return nil, err
	}

	deps := Deps{Config: cfg, Pool: pool, Content: content, Index: index}

	verifier := auth.NewVerifier(cfg.BotToken, cfg.AllowedBotID, cfg.InitDataMaxAge)
	authService := auth.NewService(pool, index, content, cfg.SessionTTL)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			httpapi.WriteError(w, httpapi.Internal(err))
			return
		}
		httpapi.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	auth.NewHandler(verifier, authService).Register(mux)
	if cfg.DevInsecureKeys {
		// Local development only: lets the Mini App be opened in a browser.
		auth.NewDevHandler(authService).Register(mux)
	}
	user.NewHandler(user.NewService(pool), authService).Register(mux)
	social.NewHandler(social.NewService(pool, content), authService).Register(mux)
	chat.NewHandler(chat.NewService(pool, content), authService, chat.NewHub()).Register(mux)

	docClient := doc.NewClient(cfg.DocAPIURL, cfg.DocAPIToken)
	docService := doc.NewService(pool, docClient)
	doc.NewHandler(docService, authService).Register(mux)

	dayClient := day.NewClient(cfg.DayAPIURL, cfg.DayAPIToken)
	dayService := day.NewService(pool, dayClient)
	day.NewHandler(dayService, authService).Register(mux)

	notesService := notes.NewService(pool, content)
	notes.NewHandler(notesService, authService).Register(mux)

	split.NewHandler(split.NewService(pool, content), authService).Register(mux)

	pulse.NewHandler(pulse.NewService(pool, content), authService).Register(mux)

	integration.NewHandler(
		integration.NewService(pool, docService, dayService, notesService, authService),
		authService,
	).Register(mux)

	// Unknown API routes answer JSON, never the SPA shell.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		httpapi.WriteError(w, httpapi.NotFound("Маршрут не найден"))
	})

	_ = ctx
	_ = deps
	return mux, nil
}
