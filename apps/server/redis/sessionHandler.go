package sessions

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionData struct {
	Email string
	Name  string
}

func sessKey(sessionID string) string { return "sess:" + sessionID }

func (s *Store) SaveSession(ctx context.Context, sessionID string, data SessionData, ttl time.Duration) error {
	key := sessKey(sessionID)
	println("REDIS KEY SAVED:\n", key)

	if err := s.rdb.HSet(ctx, key, map[string]interface{}{
		"email": data.Email,
		"name":  data.Name,
	}).Err(); err != nil {
		return err
	}
	// sets TTL
	return s.rdb.Expire(ctx, key, ttl).Err()
}

func (s *Store) GetSession(ctx context.Context, sessionID string) (*SessionData, error) {
	key := sessKey(sessionID)
	println("FETCHING REDIS KEY:\n", key)

	m, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		println("Redis error (sessionHandler.go):", err)
		return nil, err
	}
	if len(m) == 0 {
		println("Redis entry not found / expired (sessionHandler.go)")
		return nil, redis.Nil // not found / expired
	}
	return &SessionData{Email: m["email"], Name: m["name"]}, nil
}

// refresh TTL on successful login
func (s *Store) RefreshSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	return s.rdb.Expire(ctx, sessKey(sessionID), ttl).Err()
}

func (s *Store) DeleteSession(ctx context.Context, sessionID string) error {
	return s.rdb.Del(ctx, sessKey(sessionID)).Err()
}
