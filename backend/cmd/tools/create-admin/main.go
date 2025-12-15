package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	email := flag.String("email", "", "Admin email (required)")
	name := flag.String("name", "", "Admin name (required)")
	password := flag.String("password", "", "Admin password (required)")
	flag.Parse()

	if *email == "" || *name == "" || *password == "" {
		flag.Usage()
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
	defer conn.Close(ctx)

	// Check if email already exists
	var exists bool
	if err := conn.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM platform_admins WHERE email = $1)",
		*email,
	).Scan(&exists); err != nil {
		log.Fatalf("failed to check email: %v", err)
	}
	if exists {
		log.Fatalf("admin with email %s already exists", *email)
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	// Insert admin
	id := uuid.New()
	if _, err := conn.Exec(ctx,
		`INSERT INTO platform_admins (id, email, password_hash, name, is_active)
		 VALUES ($1, $2, $3, $4, true)`,
		id, *email, string(hash), *name,
	); err != nil {
		log.Fatalf("failed to create admin: %v", err)
	}

	fmt.Printf("Admin created: %s (%s)\n", *name, *email)
	fmt.Printf("ID: %s\n", id)
}
