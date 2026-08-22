package main_test

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"safetyplatform/internal/handler"
	"safetyplatform/internal/middleware"
	"safetyplatform/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func waitAuditError(t *testing.T, recorder *middleware.AuditRecorder) error {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if err := recorder.LastError(); err != nil {
			return err
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("audit worker did not publish its error")
	return nil
}

func TestAuditQueueFullErrorClassified(t *testing.T) {
	gate := make(chan struct{})
	started := make(chan struct{}, 1)
	recorder := middleware.NewAuditRecorder(1, func(entry *model.AuditLog) error {
		started <- struct{}{}
		<-gate
		return nil
	})
	if err := recorder.Record(&model.AuditLog{Action: "first"}); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := recorder.Record(&model.AuditLog{Action: "second"}); err != nil {
		t.Fatal(err)
	}
	err := recorder.Record(&model.AuditLog{Action: "third"})
	close(gate)
	if !errors.Is(err, middleware.ErrAuditQueueFull) {
		t.Fatalf("queue error is not classifiable: %v", err)
	}
}

func TestAuditDatabaseErrorUnwraps(t *testing.T) {
	dbErr := errors.New("database offline")
	recorder := middleware.NewAuditRecorder(1, func(entry *model.AuditLog) error { return dbErr })
	if err := recorder.Record(&model.AuditLog{Action: "update"}); err != nil {
		t.Fatal(err)
	}
	err := waitAuditError(t, recorder)
	if !errors.Is(err, middleware.ErrAuditDatabase) || !errors.Is(err, dbErr) {
		t.Fatalf("database error chain = %v", err)
	}
}

func TestAuditDetailSurvivesAsyncWrite(t *testing.T) {
	gate := make(chan struct{})
	started := make(chan struct{}, 1)
	stored := make(chan string, 1)
	recorder := middleware.NewAuditRecorder(1, func(entry *model.AuditLog) error {
		started <- struct{}{}
		<-gate
		stored <- entry.Detail
		return nil
	})
	entry := &model.AuditLog{Action: "review", Detail: `{"decision":"approved"}`}
	if err := recorder.Record(entry); err != nil {
		t.Fatal(err)
	}
	<-started
	entry.Detail = `{"decision":"overwritten"}`
	close(gate)
	if got := <-stored; got != `{"decision":"approved"}` {
		t.Fatalf("stored detail = %s", got)
	}
}

func TestAuditListReportsDegradedState(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	recorder := middleware.NewAuditRecorder(1, func(entry *model.AuditLog) error { return errors.New("write failed") })
	if err := recorder.Record(&model.AuditLog{Action: "create"}); err != nil {
		t.Fatal(err)
	}
	_ = waitAuditError(t, recorder)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := handler.NewAuditLogHandler(db, logger, recorder)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/audit-logs", nil)
	h.List(c)
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`"delivery_degraded":true`)) {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}
