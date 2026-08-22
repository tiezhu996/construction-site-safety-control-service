package dto_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"safetyplatform/internal/constants"
	"safetyplatform/internal/handler"
	"safetyplatform/internal/model"
	"safetyplatform/internal/repository"
	"safetyplatform/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func inspectionExecutionFixture(t *testing.T, status string, itemCount int) (*gorm.DB, *repository.SafetyInspectionRepository, *service.SafetyInspectionService, *model.SafetyInspection, []model.InspectionItem) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SafetyInspection{}, &model.InspectionItem{}); err != nil {
		t.Fatal(err)
	}
	ins := &model.SafetyInspection{Name: "tower crane check", InspectionDate: time.Now(), InspectorID: 9, Status: status}
	if err := db.Create(ins).Error; err != nil {
		t.Fatal(err)
	}
	items := make([]model.InspectionItem, itemCount)
	for i := range items {
		items[i] = model.InspectionItem{InspectionID: ins.ID, ItemName: "checkpoint " + strconv.Itoa(i+1)}
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.NewSafetyInspectionRepository(db)
	itemRepo := repository.NewInspectionItemRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.NewSafetyInspectionService(db, repo, itemRepo, repository.NewUserRepository(db), logger)
	return db, repo, svc, ins, items
}

func TestInspectionPartialMovesInProgress(t *testing.T) {
	_, _, svc, ins, items := inspectionExecutionFixture(t, constants.InspectionScheduled, 2)
	got, err := svc.Execute(ins.ID, []model.InspectionItem{{ID: items[0].ID, Passed: true, Remark: "ok"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != constants.InspectionInProgress {
		t.Fatalf("partial execution status = %q", got.Status)
	}
}

func TestInspectionCompletionFromProgress(t *testing.T) {
	_, _, svc, ins, items := inspectionExecutionFixture(t, constants.InspectionInProgress, 2)
	updates := []model.InspectionItem{{ID: items[0].ID, Passed: true}, {ID: items[1].ID, Passed: true}}
	got, err := svc.Execute(ins.ID, updates)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != constants.InspectionCompleted || got.PassedCount != 2 {
		t.Fatalf("completed execution = status %q passed %d", got.Status, got.PassedCount)
	}
}

func TestInspectionTerminalCannotReopen(t *testing.T) {
	db, repo, _, ins, _ := inspectionExecutionFixture(t, constants.InspectionCompleted, 1)
	stale := *ins
	stale.Status = constants.InspectionInProgress
	if err := repo.UpdateStateTx(db, &stale, constants.InspectionScheduled); err == nil {
		t.Fatal("stale scheduled state reopened a completed inspection")
	}
	var stored model.SafetyInspection
	if err := db.First(&stored, ins.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != constants.InspectionCompleted {
		t.Fatalf("terminal status changed to %q", stored.Status)
	}
}

func TestInspectionReportVisibleAtTerminal(t *testing.T) {
	_, _, svc, ins, _ := inspectionExecutionFixture(t, constants.InspectionCompleted, 1)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := handler.NewSafetyInspectionHandler(svc, logger)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/inspections/report", nil)
	c.Params = []gin.Param{{Key: "id", Value: strconv.FormatUint(ins.ID, 10)}}
	h.Report(c)
	if w.Code != http.StatusOK {
		t.Fatalf("completed report status = %d, body = %s", w.Code, w.Body.String())
	}
}
