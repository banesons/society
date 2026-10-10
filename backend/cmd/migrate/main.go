package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"chs/backend/internal/config"
	"chs/backend/internal/database"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required to run migrations")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	targetMigration := "20261011000000_create_societies_schema.sql"
	if len(os.Args) > 1 && os.Args[1] != "" {
		targetMigration = os.Args[1]
	}

	migrationFile := filepath.Join("..", "supabase", "migrations", targetMigration)
	sqlBytes, err := os.ReadFile(migrationFile)
	if err != nil {
		// Try relative to workspace root
		migrationFile = filepath.Join("supabase", "migrations", targetMigration)
		sqlBytes, err = os.ReadFile(migrationFile)
		if err != nil {
			log.Fatalf("failed to read migration file %s: %v", targetMigration, err)
		}
	}

	fmt.Printf("Applying migration: %s\n", migrationFile)
	_, err = db.Pool().Exec(ctx, string(sqlBytes))
	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	fmt.Println("Migration applied successfully!")
}
