package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/migrations"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: migrator <command>")
		fmt.Println("Commands: up, down, status, version")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := goose.OpenDBWithDriver("pgx", cfg.Database.URL)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close database", "error", err)
		}
	}()

	goose.SetBaseFS(migrations.FS)

	if err := goose.RunContext(context.Background(), os.Args[1], db, ".", os.Args[2:]...); err != nil {
		slog.Error("migration failed", "command", os.Args[1], "error", err)
		os.Exit(1)
	}
}
