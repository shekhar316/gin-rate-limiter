package limiter

import (
	"context"
	"time"
)

// The interface for rate-limiting storage.
// It provides funcationality to interact with a storage backend (in-memory or Redis).
type Store interface {
	// Increments the counter for a given key and returns the new value.
	Increment(ctx context.Context, key string, window time.Duration) (int, error)

	// Returns the current value of a counter for a given key.
	Get(ctx context.Context, key string) (int, error)

	// Returns the current value and remaining time for a counter.
	GetWithTime(ctx context.Context, key string) (int, time.Duration, error)

	// Adds a timestamp to a list (for sliding window log).
	AddToList(ctx context.Context, key string, timestamp int64) error

	// Returns the length of a list.
	GetListLength(ctx context.Context, key string) (int, error)

	// Removes old entries from a list.
	TrimList(ctx context.Context, key string, minTimestamp int64) error

	// Take a token from a bucket.
	TakeToken(ctx context.Context, key string, rate float64, burst int, now int64) (bool, error)

	// Add a request to a queue 
	Enqueue(ctx context.Context, key string, burst int, now int64) (bool, error)

	// Processes the queue at a given rate.
	Dequeue(ctx context.Context, key string, rate float64, now int64)
}
