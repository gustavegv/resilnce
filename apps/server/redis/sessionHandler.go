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

func (s *Store) GetOrInitAICalls(ctx context.Context, sessionID string) (int64, error) {
	key := sessKey(sessionID)

	exists, err := s.rdb.Exists(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if exists == 0 {
		return 0, redis.Nil
	}

	// get or create (init with 0)
	val, err := s.rdb.HIncrBy(ctx, key, "aiCalls", 0).Result()
	if err != nil {
		return 0, err
	}
	return val, nil
}

func (s *Store) IncrAICalls(ctx context.Context, sessionID string, delta int64) (int64, error) {
	key := sessKey(sessionID)

	// check ttl
	exists, err := s.rdb.Exists(ctx, key).Result()
	if err != nil {
		return -1, err
	}
	if exists == 0 {
		return -2, redis.Nil
	}

	// get or create (incr with delta)
	val, err := s.rdb.HIncrBy(ctx, key, "aiCalls", delta).Result()
	if err != nil {
		return -3, err
	}
	return val, nil
}
