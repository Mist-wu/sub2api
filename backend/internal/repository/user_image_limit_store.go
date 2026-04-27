package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Mist-wu/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const userImageLimitRedisPrefix = "sub2api:user-image"

var userImageLimitDecrementScript = redis.NewScript(`
local current = tonumber(redis.call("GET", KEYS[1]) or "0")
if current <= 1 then
	return redis.call("DEL", KEYS[1])
end
return redis.call("DECR", KEYS[1])
`)

type userImageLimitStore struct {
	rdb *redis.Client

	mu          sync.Mutex
	dailyCounts map[string]int
	active      map[int64]int
}

// NewUserImageLimitStore creates a Redis-backed image limit store.
func NewUserImageLimitStore(rdb *redis.Client) service.UserImageLimitStore {
	return &userImageLimitStore{
		rdb:         rdb,
		dailyCounts: make(map[string]int),
		active:      make(map[int64]int),
	}
}

func (s *userImageLimitStore) ReserveDaily(ctx context.Context, userID int64, day string, limit int, ttl time.Duration) error {
	if s == nil {
		return nil
	}
	if limit <= 0 {
		return nil
	}
	key := fmt.Sprintf("%s:daily:%s:%d", userImageLimitRedisPrefix, day, userID)
	if s.rdb != nil {
		count, err := s.rdb.Incr(ctx, key).Result()
		if err != nil {
			return err
		}
		if count == 1 && ttl > 0 {
			_ = s.rdb.Expire(ctx, key, ttl).Err()
		}
		if count > int64(limit) {
			_ = decrementUserImageLimitKey(ctx, s.rdb, key)
			return service.ErrUserImageDailyLimit
		}
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dailyCounts[key] >= limit {
		return service.ErrUserImageDailyLimit
	}
	s.dailyCounts[key]++
	return nil
}

func (s *userImageLimitStore) AcquireConcurrency(ctx context.Context, userID int64, limit int, ttl time.Duration) (func(), error) {
	if s == nil || limit <= 0 {
		return func() {}, nil
	}
	key := fmt.Sprintf("%s:active:%d", userImageLimitRedisPrefix, userID)
	if s.rdb != nil {
		count, err := s.rdb.Incr(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		if ttl > 0 {
			_ = s.rdb.Expire(ctx, key, ttl).Err()
		}
		if count > int64(limit) {
			_ = decrementUserImageLimitKey(ctx, s.rdb, key)
			return nil, service.ErrUserImageConcurrency
		}
		var once sync.Once
		return func() {
			once.Do(func() {
				_ = decrementUserImageLimitKey(context.Background(), s.rdb, key)
			})
		}, nil
	}

	s.mu.Lock()
	if s.active[userID] >= limit {
		s.mu.Unlock()
		return nil, service.ErrUserImageConcurrency
	}
	s.active[userID]++
	s.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.active[userID] > 0 {
				s.active[userID]--
			}
		})
	}, nil
}

func decrementUserImageLimitKey(ctx context.Context, rdb *redis.Client, key string) error {
	if rdb == nil {
		return nil
	}
	return userImageLimitDecrementScript.Run(ctx, rdb, []string{key}).Err()
}
