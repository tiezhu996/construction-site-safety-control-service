package router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"safetyplatform/internal/handler"
	"safetyplatform/internal/model"
	"safetyplatform/internal/repository"
	"safetyplatform/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type batchUpdater struct {
	gate    <-chan struct{}
	started chan<- struct{}
}

func (u *batchUpdater) UpdateBatchItem(ctx context.Context, item *model.InspectionItem) error {
	if u.started != nil {
		u.started <- struct{}{}
	}
	if u.gate != nil {
		select {
		case <-u.gate:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func TestItemBatchWaitsForAllWorkers(t *testing.T) {
	gate := make(chan struct{})
	started := make(chan struct{}, 3)
	svc := service.NewInspectionItemBatchService(&batchUpdater{gate: gate, started: started})
	done := make(chan []model.InspectionItem, 1)
	go func() {
		items, _ := svc.Process(context.Background(), []model.InspectionItem{{ID: 1}, {ID: 2}, {ID: 3}})
		done <- items
	}()
	for i := 0; i < 3; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("worker did not start")
		}
	}
	select {
	case got := <-done:
		close(gate)
		t.Fatalf("batch returned before workers completed: %d results", len(got))
	case <-time.After(30 * time.Millisecond):
	}
	close(gate)
	select {
	case got := <-done:
		if len(got) != 3 {
			t.Fatalf("results = %d", len(got))
		}
	case <-time.After(time.Second):
		t.Fatal("batch did not finish")
	}
}

func TestItemBatchErrorPathDoesNotHang(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.InspectionItem{}); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewInspectionItemRepository(db)
	if err := repo.UpdateBatchItem(context.Background(), &model.InspectionItem{ID: 999}); err == nil {
		t.Fatal("missing item update was reported as successful")
	}
}

func TestItemBatchChannelClosesOnce(t *testing.T) {
	a := model.InspectionItem{ID: 4, InspectionID: 8}
	b := model.InspectionItem{ID: 5, InspectionID: 8}
	if a.BatchKey() == b.BatchKey() {
		t.Fatalf("two batch results share key %q", a.BatchKey())
	}
}

func TestItemBatchConcurrentUpdatesRaceFree(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	batch := service.NewInspectionItemBatchService(&batchUpdater{})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := handler.NewInspectionItemHandler(nil, logger, batch)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for n := 0; n < 16; n++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start
			payload, _ := json.Marshal([]model.InspectionItem{{ID: uint64(id + 1), InspectionID: 3}})
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/inspection-items/batch", bytes.NewReader(payload))
			c.Request.Header.Set("Content-Type", "application/json")
			h.Batch(c)
			if w.Code != http.StatusOK {
				t.Errorf("status = %d", w.Code)
			}
		}(n)
	}
	close(start)
	wg.Wait()
}
