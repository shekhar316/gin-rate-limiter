package limiter

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type memoryEntry struct {
	value      int
	lastSeen   time.Time
	timestamps []int64
	limiter    *rate.Limiter
	queue      []int64
}

type MemoryStore struct {
	visitors map[string]*memoryEntry
	mu       sync.Mutex
}

func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		visitors: make(map[string]*memoryEntry),
	}
	go store.cleanup()
	return store
}

func (s *MemoryStore) getVisitor(key string) *memoryEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, exists := s.visitors[key]
	if !exists {
		v = &memoryEntry{}
		s.visitors[key] = v
	}
	v.lastSeen = time.Now()
	return v
}

func (s *MemoryStore) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
	v := s.getVisitor(key)
	if time.Since(v.lastSeen) > window {
		v.value = 0 
	}
	v.value++
	return v.value, nil
}

func (s *MemoryStore) AddToList(ctx context.Context, key string, timestamp int64) error {
	v := s.getVisitor(key)
	v.timestamps = append(v.timestamps, timestamp)
	return nil
}

func (s *MemoryStore) GetListLength(ctx context.Context, key string) (int, error) {
	v := s.getVisitor(key)
	return len(v.timestamps), nil
}

func (s *MemoryStore) TrimList(ctx context.Context, key string, minTimestamp int64) error {
	v := s.getVisitor(key)
	var validTimestamps []int64
	for _, ts := range v.timestamps {
		if ts >= minTimestamp {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	v.timestamps = validTimestamps
	return nil
}

func (s *MemoryStore) TakeToken(ctx context.Context, key string, r float64, b int, now int64) (bool, error) {
	v := s.getVisitor(key)
	if v.limiter == nil {
		v.limiter = rate.NewLimiter(rate.Limit(r), b)
	}
	return v.limiter.Allow(), nil
}

func (s *MemoryStore) Enqueue(ctx context.Context, key string, burst int, now int64) (bool, error) {
	v := s.getVisitor(key)
	if len(v.queue) >= burst {
		return false, nil 
	}
	v.queue = append(v.queue, now)
	return true, nil
}

func (s *MemoryStore) Dequeue(ctx context.Context, key string, r float64, now int64) {
	v := s.getVisitor(key)
	if len(v.queue) == 0 {
		return
	}

	interval := time.Duration(1e9/r) * time.Nanosecond
	if time.Since(time.Unix(0, v.queue)) >= 0 { 
		v.queue = v.queue[1:]
	}
}

func (s *MemoryStore) Get(ctx context.Context, key string) (int, error) {
	v := s.getVisitor(key)
	return v.value, nil
}

func (s *MemoryStore) GetWithTime(ctx context.Context, key string) (int, time.Duration, error) {
	return s.Get(ctx, key)
}


func (s *MemoryStore) cleanup() {
	for {
		time.Sleep(1 * time.Minute)
		s.mu.Lock()
		for key, v := range s.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(s.visitors, key)
			}
		}
		s.mu.Unlock()
	}
}
