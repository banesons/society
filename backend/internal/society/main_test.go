package society

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"chs/backend/internal/config"
	"chs/backend/internal/database"
	"chs/backend/internal/testutil"
)

func TestMain(m *testing.M) {
	cfg := config.Load()
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		db, err := database.Connect(ctx, cfg.DatabaseURL)
		if err == nil {
			_ = testutil.CleanExtraSocieties(ctx, db.Pool())
			db.Close()
		}
		cancel()
	}

	code := m.Run()

	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		db, err := database.Connect(ctx, cfg.DatabaseURL)
		if err == nil {
			if err := testutil.CleanExtraSocieties(ctx, db.Pool()); err != nil {
				fmt.Printf("[TestMain society] Warning: Failed to clean extra societies: %v\n", err)
			}
			db.Close()
		}
		cancel()
	}

	os.Exit(code)
}
