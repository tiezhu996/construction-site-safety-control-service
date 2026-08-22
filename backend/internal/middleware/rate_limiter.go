package middleware

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"safetyplatform/internal/constants"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	tokens   int
	lastFill time.Time
}

// RateLimiter 基于 IP 的令牌桶限流。
type RateLimiter struct {
	mu       sync.Mutex
	perMin   int
	buckets  map[string]*bucket
	capacity int
	cleaners atomic.Int32
	once     sync.Once
}

// NewRateLimiter 构造限流器。
func NewRateLimiter(perMin int) *RateLimiter {
	if perMin <= 0 {
		perMin = 120
	}
	return &RateLimiter{perMin: perMin, buckets: make(map[string]*bucket), capacity: perMin}
}

// Limit 返回限流中间件。
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.take(c.ClientIP(), time.Now()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": constants.CodeTooManyRequests, "message": constants.MsgTooManyRequests, "data": nil})
			return
		}
		c.Next()
	}
}

func (rl *RateLimiter) take(ip string, now time.Time) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.buckets[ip]
	if !ok {
		b = &bucket{tokens: rl.capacity, lastFill: now}
		rl.buckets[ip] = b
	}
	if refill := int(now.Sub(b.lastFill).Minutes()) * rl.perMin; refill > 0 {
		b.tokens += refill
		if b.tokens > rl.capacity {
			b.tokens = rl.capacity
		}
		b.lastFill = now
	}
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

// PruneIdleBefore 清理截止时间前已经补满的空闲 bucket。
// 仅回收已补满（tokens >= capacity）的空闲桶，避免删除仍持有
// 限流状态的活跃桶——删除活跃桶会让客户端拿到全新的满桶从而绕过限流。
func (rl *RateLimiter) PruneIdleBefore(cutoff time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for ip, b := range rl.buckets {
		if b.lastFill.Before(cutoff) && b.tokens >= rl.capacity {
			delete(rl.buckets, ip)
		}
	}
}

// BucketCount 返回当前 bucket 数。
func (rl *RateLimiter) BucketCount() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return len(rl.buckets)
}

// StartCleanup 启动后台清理器。多次调用仅启动一个清理 goroutine，
// 保证 ActiveCleaners 与实际运行数量一致。
func (rl *RateLimiter) StartCleanup(ctx context.Context, interval time.Duration) {
	rl.once.Do(func() {
		rl.cleaners.Add(1)
		go func() {
			defer rl.cleaners.Add(-1)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case now := <-ticker.C:
					rl.PruneIdleBefore(now.Add(-2 * interval))
				}
			}
		}()
	})
}

// ActiveCleaners 返回正在运行的清理器数量。
func (rl *RateLimiter) ActiveCleaners() int {
	return int(rl.cleaners.Load())
}
