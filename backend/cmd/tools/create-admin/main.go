package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"golang.org/x/crypto/bcrypt"
)

// SEC-007: bcrypt cost 12 вместо DefaultCost (10)
const bcryptCost = 12

func main() {
	email := flag.String("email", os.Getenv("ADMIN_EMAIL"), "Admin email")
	name := flag.String("name", os.Getenv("ADMIN_NAME"), "Admin name")
	password := flag.String("password", os.Getenv("ADMIN_PASSWORD"), "Admin password")
	flag.Parse()

	if *email == "" || *name == "" || *password == "" {
		fmt.Println("Usage: create-admin --email=... --name=... --password=...")
		fmt.Println("Or set ADMIN_EMAIL, ADMIN_NAME, ADMIN_PASSWORD env vars")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer func() {
		if err := conn.Close(ctx); err != nil {
			log.Printf("failed to close connection: %v", err)
		}
	}()

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcryptCost)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	// Удаляем всех существующих админов и создаём нового с актуальными данными
	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Fatalf("failed to begin transaction: %v", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			log.Printf("failed to rollback: %v", err)
		}
	}()

	tag, err := tx.Exec(ctx, "DELETE FROM platform_admins")
	if err != nil {
		log.Fatalf("failed to delete old admins: %v", err)
	}
	if tag.RowsAffected() > 0 {
		fmt.Printf("Removed %d old admin(s)\n", tag.RowsAffected())
	}

	id := uuid.New()
	if _, err := tx.Exec(ctx,
		`INSERT INTO platform_admins (id, email, password_hash, name, is_active)
		 VALUES ($1, $2, $3, $4, true)`,
		id, *email, string(hash), *name,
	); err != nil {
		log.Fatalf("failed to create admin: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("failed to commit: %v", err)
	}

	fmt.Printf("Admin ready: %s (%s)\n", *name, *email)
}
