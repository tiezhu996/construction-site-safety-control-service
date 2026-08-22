package middleware_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"safetyplatform/internal/config"
	"safetyplatform/internal/handler"
	"safetyplatform/internal/middleware"
	"safetyplatform/internal/router"

	"github.com/gin-gonic/gin"
)

func limiterEngine(rl *middleware.RateLimiter) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(rl.Limit())
	r.GET("/probe", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r
}

func hitLimiter(r http.Handler, ip string) int {
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.RemoteAddr = ip + ":3210"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec.Code
}

func TestLimiterConcurrentBudget(t *testing.T) {
	const capacity = 32
	rl := middleware.NewRateLimiter(capacity)
	engine := limiterEngine(rl)
	start := make(chan struct{})
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < capacity*2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if hitLimiter(engine, "10.8.0.1") == http.StatusNoContent {
				accepted.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if got := int(accepted.Load()); got != capacity {
		t.Fatalf("accepted=%d want=%d", got, capacity)
	}
}

func TestLimiterCleanupKeepsActiveBucket(t *testing.T) {
	rl := middleware.NewRateLimiter(2)
	engine := limiterEngine(rl)
	start := make(chan struct{})
	results := make(chan int, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- hitLimiter(engine, "10.8.0.2")
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	for status := range results {
		if status != http.StatusNoContent {
			t.Fatalf("active request was rejected with status %d", status)
		}
	}
	rl.PruneIdleBefore(time.Now().Add(time.Hour))
	if rl.BucketCount() != 1 {
		t.Fatalf("active bucket was pruned, count=%d", rl.BucketCount())
	}
}

func TestLimiterRejectsAfterCapacity(t *testing.T) {
	rl := middleware.NewRateLimiter(2)
	engine := limiterEngine(rl)
	start := make(chan struct{})
	var accepted atomic.Int32
	var rejected atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			switch hitLimiter(engine, "10.8.0.3") {
			case http.StatusNoContent:
				accepted.Add(1)
			case http.StatusTooManyRequests:
				rejected.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if got := accepted.Load(); got != 2 {
		t.Fatalf("accepted=%d want=2", got)
	}
	if got := rejected.Load(); got != 6 {
		t.Fatalf("rejected=%d want=6", got)
	}
}

func TestLimiterParallelIsolation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := config.Load()
	r := router.New(cfg, nil, logger,
		new(handler.UserHandler), new(handler.SafetyIncidentHandler),
		new(handler.SafetyInspectionHandler), new(handler.InspectionItemHandler),
		new(handler.SafetyTrainingHandler), new(handler.WorkerCertificationHandler),
		new(handler.DashboardHandler), new(handler.UploadHandler), new(handler.AuditLogHandler))
	r.SetCleanupContext(ctx)
	engines := []http.Handler{r.Setup(), r.Setup()}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			_ = hitLimiter(engines[index], "10.8.1."+string(rune('1'+index)))
		}(i)
	}
	close(start)
	wg.Wait()
	if got := r.ActiveLimiterCleaners(); got != 1 {
		t.Fatalf("router started %d limiter cleaners", got)
	}
}
