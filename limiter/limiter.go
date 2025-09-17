package limiter

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Returns the client identifier (IP)
func getIdentifier(c *gin.Context) string {
	return c.ClientIP()
}

// Implements the fixed window algorithm.
func FixedWindowLimiter(store Store, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("fw:%s:%d", getIdentifier(c), time.Now().Unix()/int64(window.Seconds()))

		count, err := store.Increment(c.Request.Context(), key, window)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(limit-count))

		if count > limit {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "Too many requests"})
			return
		}
		c.Next()
	}
}

// Implements the token bucket algorithm.
func TokenBucketLimiter(store Store, rate float64, burst int) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("tb:%s", getIdentifier(c))

		ok, err := store.TakeToken(c.Request.Context(), key, rate, burst, time.Now().Unix())
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		if !ok {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "Too many requests"})
			return
		}
		c.Next()
	}
}

// Implements the sliding window log algorithm.
func SlidingWindowLogLimiter(store Store, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now().UnixNano()
		key := fmt.Sprintf("swl:%s", getIdentifier(c))

		ctx := c.Request.Context()
		_ = store.TrimList(ctx, key, now-window.Nanoseconds())
		_ = store.AddToList(ctx, key, now)
		count, _ := store.GetListLength(ctx, key)

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(limit-count))

		if count > limit {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "Too many requests"})
			return
		}
		c.Next()
	}
}

// Implements the sliding window counter algorithm.
func SlidingWindowCounterLimiter(store Store, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now()
		currentWindow := now.Unix() / int64(window.Seconds())
		prevWindow := currentWindow - 1

		currentKey := fmt.Sprintf("swc:%s:%d", getIdentifier(c), currentWindow)
		prevKey := fmt.Sprintf("swc:%s:%d", getIdentifier(c), prevWindow)

		ctx := c.Request.Context()
		prevCount, _ := store.Get(ctx, prevKey)
		currentCount, err := store.Increment(ctx, currentKey, window*2) 
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		percentage := float64(now.Unix()%int64(window.Seconds())) / float64(window.Seconds())
		rate := float64(prevCount)*(1-percentage) + float64(currentCount)

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%.0f", float64(limit)-rate))

		if int(rate) >= limit {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "Too many requests"})
			return
		}

		c.Next()
	}
}

func LeakyBucketLimiter(store Store, rate float64, burst int) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("lb:%s", getIdentifier(c))
		now := time.Now().UnixNano()

		ctx := c.Request.Context()
		store.Dequeue(ctx, key, rate, now) // "Process" one item
		ok, err := store.Enqueue(ctx, key, burst, now)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		if !ok {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "Too many requests"})
			return
		}
		c.Next()
	}
}
