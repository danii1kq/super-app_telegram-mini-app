package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/config"
	"github.com/danii1kq/srunal/internal/platform/postgres"
)

// migrate applies the SQL files in db/migrations in filename order.
//
// shortcut: golang-migrate is not wired yet; a schema_migrations table keeps
// this idempotent. Switch when down-migrations are needed in CI.
func main() {
	dir := flag.String("dir", "db/migrations", "migrations directory")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.LoadDatabase()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    text PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		slog.Error("create schema_migrations", "err", err)
		os.Exit(1)
	}

	files, err := migrationFiles(*dir)
	if err != nil {
		slog.Error("read migrations", "err", err)
		os.Exit(1)
	}

	for _, name := range files {
		version := strings.TrimSuffix(name, ".up.sql")

		var applied bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version,
		).Scan(&applied); err != nil {
			slog.Error("check migration", "version", version, "err", err)
			os.Exit(1)
		}
		if applied {
			continue
		}

		body, err := os.ReadFile(filepath.Join(*dir, name))
		if err != nil {
			slog.Error("read migration", "file", name, "err", err)
			os.Exit(1)
		}

		// One transaction per migration: a failure leaves the previous version.
		err = pool.InTx(ctx, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, string(body)); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version)
			return err
		})
		if err != nil {
			slog.Error("apply migration", "version", version, "err", err)
			os.Exit(1)
		}
		slog.Info("migration applied", "version", version)
	}

	fmt.Println("migrations up to date")
}

func migrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)
	return files, nil
}
