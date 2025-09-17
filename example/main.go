package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/shekhar316/gin-rate-limiter/limiter" // Adjust import path
)

func main() {
	router := gin.Default()

	memStore := limiter.NewMemoryStore()
	inMemoryGroup := router.Group("/memory")
	{
		// 10 requests per minute
		inMemoryGroup.GET("/fixed", limiter.FixedWindowLimiter(memStore, 10, time.Minute), successHandler)
		// 2 requests/sec, burst of 5
		inMemoryGroup.GET("/token", limiter.TokenBucketLimiter(memStore, 2, 5), successHandler)
	}

	// Redis Store Example
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	redisStore := limiter.NewRedisStore(rdb)
	redisGroup := router.Group("/redis")
	{
		// 100 requests per hour
		redisGroup.GET("/sliding-log", limiter.SlidingWindowLogLimiter(redisStore, 100, time.Hour), successHandler)
		// 50 requests per 10 minutes
		redisGroup.GET("/sliding-counter", limiter.SlidingWindowCounterLimiter(redisStore, 50, 10*time.Minute), successHandler)
	}

	router.Run(":8080")
}

func successHandler(c *gin.Context) {
	c.String(http.StatusOK, "Request allowed!")
}
