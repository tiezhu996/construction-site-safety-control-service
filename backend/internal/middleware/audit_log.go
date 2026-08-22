package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"safetyplatform/internal/constants"
	"safetyplatform/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	ErrAuditQueueFull = errors.New("audit queue full")
	ErrAuditDatabase  = errors.New("audit database write failed")
)

// AuditRecorder owns the bounded asynchronous audit queue.
type AuditRecorder struct {
	queue   chan *model.AuditLog
	persist func(*model.AuditLog) error
	mu      sync.RWMutex
	lastErr error
}

func NewAuditRecorder(capacity int, persist func(*model.AuditLog) error) *AuditRecorder {
	r := &AuditRecorder{queue: make(chan *model.AuditLog, capacity), persist: persist}
	go r.run()
	return r
}

func (r *AuditRecorder) run() {
	for entry := range r.queue {
		entry.DeliveryState = "persisting"
		if err := r.persist(entry); err != nil {
			r.mu.Lock()
			r.lastErr = fmt.Errorf("%v: %v", ErrAuditDatabase, err)
			r.mu.Unlock()
			continue
		}
		entry.DeliveryState = "persisted"
	}
}

func (r *AuditRecorder) Record(entry *model.AuditLog) error {
	entry.DeliveryState = "queued"
	select {
	case r.queue <- entry.Snapshot():
		return nil
	default:
		return fmt.Errorf("audit delivery: %v", ErrAuditQueueFull)
	}
}

func (r *AuditRecorder) LastError() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastErr
}

// AuditLog 操作审计日志中间件。
func AuditLog(db *gorm.DB, logger *slog.Logger) gin.HandlerFunc {
	recorder := NewAuditRecorder(128, func(entry *model.AuditLog) error {
		return db.Create(entry).Error
	})
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}
		c.Next()
		if c.Writer.Status() >= 400 {
			return
		}
		path := c.FullPath()
		entityType := "unknown"
		seg := strings.Split(strings.TrimPrefix(path, "/api/v1/"), "/")
		if len(seg) > 0 && seg[0] != "" {
			entityType = seg[0]
		}
		detail := map[string]any{"method": c.Request.Method, "path": path}
		if b, ok := c.Get("audit_detail"); ok {
			detail["body"] = b
		}
		raw, _ := json.Marshal(detail)
		entry := &model.AuditLog{
			OperatorID: GetUserID(c), OperatorName: GetPhone(c),
			Action: c.Request.Method, EntityType: entityType,
			EntityID: c.Param("id"), Detail: string(raw), IP: c.ClientIP(), CreatedAt: time.Now(),
		}
		if err := recorder.Record(entry); err != nil {
			logger.Error(constants.LogAuditWriteFailed, "error", err.Error())
		}
	}
}
