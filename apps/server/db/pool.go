// db/pool.go
package db

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool() *pgxpool.Pool {
	connectionDetails := os.Getenv("DATABASE_URL")
	if connectionDetails == "" {
		log.Fatal("DATABASE_URL not set")
	}

	config, err := pgxpool.ParseConfig(connectionDetails)
	if err != nil {
		log.Fatalf("Failed to parse DATABASE_URL: %v", err)
	}

	config.MaxConns = 8
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}

	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctxPing); err != nil {
		log.Fatalf("Cannot reach Supabase: %v", err)
	}

	log.Println("Connected to Supabase via Session Pooler.")
	return pool
}
