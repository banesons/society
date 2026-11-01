package main

import (
	"context"
	"fmt"
	"os"

	"chs/backend/internal/config"
	"chs/backend/internal/database"
	"chs/backend/internal/testutil"
)

func main() {
	cfg := config.Load()
	db, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		fmt.Printf("Connect error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()
	conn := db.Pool()

	fmt.Println("=== ENFORCING STRICT TWO-SOCIETY CANONICAL BASELINE ===")
	if err := testutil.CleanExtraSocieties(ctx, conn); err != nil {
		fmt.Printf("Error cleaning extra societies: %v\n", err)
		os.Exit(1)
	}

	var count int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM societies").Scan(&count)
	fmt.Printf("\nSuccess: Database restored to exactly %d canonical societies.\n", count)

	rows, _ := conn.Query(ctx, "SELECT id, society_code, name, status FROM societies ORDER BY name")
	for rows.Next() {
		var id, code, name, status string
		_ = rows.Scan(&id, &code, &name, &status)
		fmt.Printf("  - %s: %s (%s) [%s]\n", id, name, code, status)
	}
	rows.Close()
}
