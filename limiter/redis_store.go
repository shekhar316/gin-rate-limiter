package limiter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

func (s *RedisStore) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
	val, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if val == 1 {
		s.client.Expire(ctx, key, window)
	}
	return int(val), nil
}

func (s *RedisStore) AddToList(ctx context.Context, key string, timestamp int64) error {
	member := redis.Z{Score: float64(timestamp), Member: fmt.Sprintf("%d", timestamp)}
	return s.client.ZAdd(ctx, key, &member).Err()
}

func (s *RedisStore) GetListLength(ctx context.Context, key string) (int, error) {
	val, err := s.client.ZCard(ctx, key).Result()
	return int(val), err
}

func (s *RedisStore) TrimList(ctx context.Context, key string, minTimestamp int64) error {
	return s.client.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(minTimestamp-1, 10)).Err()
}


var takeTokenScript = redis.NewScript(`
    local key = KEYS[1]
    local rate = tonumber(ARGV[1])
    local burst = tonumber(ARGV[2])
    local now = tonumber(ARGV[3])
    
    local data = redis.call('HMGET', key, 'tokens', 'last_seen')
    local tokens = tonumber(data[1])
    local last_seen = tonumber(data[2])
    
    if tokens == nil then
        tokens = burst
        last_seen = now
    end
    
    local elapsed = now - last_seen
    tokens = tokens + elapsed * rate
    if tokens > burst then
        tokens = burst
    end
    
    if tokens >= 1 then
        tokens = tokens - 1
        redis.call('HMSET', key, 'tokens', tokens, 'last_seen', now)
        redis.call('EXPIRE', key, math.ceil(burst / rate) * 2)
        return 1
    else
        return 0
    end
`)

func (s *RedisStore) TakeToken(ctx context.Context, key string, r float64, b int, now int64) (bool, error) {
	res, err := takeTokenScript.Run(ctx, s.client, []string{key}, r, b, now).Result()
	if err != nil {
		return false, err
	}
	return res.(int64) == 1, nil
}

func (s *RedisStore) Get(ctx context.Context, key string) (int, error) {
	val, err := s.client.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}
func (s *RedisStore) GetWithTime(ctx context.Context, key string) (int, time.Duration, error) {
	pipe := s.client.Pipeline()
	getCmd := pipe.Get(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return 0, 0, err
	}
	count, _ := getCmd.Int()
	ttl, _ := ttlCmd.Result()
	return count, ttl, nil
}

func (s *RedisStore) Enqueue(ctx context.Context, key string, burst int, now int64) (bool, error) {
	length, err := s.client.RPushX(ctx, key, now).Result()
	if err != nil && err != redis.Nil {
		return false, err
	}
	if length >= int64(burst) {
		return false, nil // Full
	}
	s.client.RPush(ctx, key, now)
	return true, nil
}

func (s *RedisStore) Dequeue(ctx context.Context, key string, rate float64, now int64) {
	// Dummy implementation for Redis Store
}
