package sessions

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type Store struct {
	rdb *redis.Client
}

func NewStore() (*Store, error) {
	addr := os.Getenv("REDIS_ADDR")
	pass := os.Getenv("REDIS_PASSWORD")
	host, _, _ := net.SplitHostPort(addr)

	if addr == "" || pass == "" || host == "" {
		log.Fatal("Failed to load Redis ENV variables (sessions.go)")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     pass,
		DB:           0,
		PoolSize:     10, // total conns
		MinIdleConns: 2,
		MaxIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		MaxRetries:   2,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		panic(fmt.Errorf("redis connection failed: %w", err))
	}

	return &Store{rdb: rdb}, nil
}
