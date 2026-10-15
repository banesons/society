package main

import (
	"context"
	"fmt"
	"log"

	"chs/backend/internal/config"
	"chs/backend/internal/database"
)

func main() {
	cfg := config.Load()
	db, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Pool().Query(context.Background(), "SELECT action, count(*) FROM platform_audit_logs GROUP BY action ORDER BY count(*) DESC")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("=== DISTINCT AUDIT ACTIONS IN DATABASE ===")
	for rows.Next() {
		var a string
		var c int
		_ = rows.Scan(&a, &c)
		fmt.Printf("%-40s %d\n", a, c)
	}
}
