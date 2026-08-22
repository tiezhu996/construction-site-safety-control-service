package handler_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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

func openIncidentDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.SafetyIncident{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func incidentFixture(t *testing.T, status string) (*repository.SafetyIncidentRepository, *service.SafetyIncidentService, uint64) {
	t.Helper()
	db := openIncidentDB(t)
	userRepo := repository.NewUserRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := repository.NewSafetyIncidentRepository(db)
	incident := &model.SafetyIncident{Title: "edge barrier", Status: status, SeverityLevel: constants.SeverityMinor}
	if err := db.Create(incident).Error; err != nil {
		t.Fatal(err)
	}
	return repo, service.NewSafetyIncidentService(repo, userRepo, logger), incident.ID
}

func TestIncidentCanceledQueryStops(t *testing.T) {
	repo, _, id := incidentFixture(t, constants.IncidentReported)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := repo.FindByIDContext(ctx, id)
	if err == nil || ctx.Err() == nil {
		t.Fatalf("canceled query must stop, got %v", err)
	}
}

func TestIncidentFreshRequestSurvivesOldCancel(t *testing.T) {
	repo, _, id := incidentFixture(t, constants.IncidentReported)
	old, cancel := context.WithCancel(context.Background())
	cancel()
	_, _ = repo.FindByIDContext(old, id)
	fresh, freshCancel := context.WithTimeout(context.Background(), time.Second)
	defer freshCancel()
	if _, err := repo.FindByIDContext(fresh, id); err != nil {
		t.Fatalf("fresh request inherited old cancellation: %v", err)
	}
}

func TestIncidentHandlerPassesRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, svc, id := incidentFixture(t, constants.IncidentReported)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := handler.NewSafetyIncidentHandler(svc, logger)
	r := gin.New()
	r.GET("/incidents/:id", h.Get)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/incidents/"+formatID(id), nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("canceled HTTP request reached a successful query: %s", rec.Body.String())
	}
}

func TestIncidentRectifyHonorsDeadline(t *testing.T) {
	repo, svc, id := incidentFixture(t, constants.IncidentInvestigating)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.SubmitRectificationContext(ctx, id, "replace guardrail", nil)
	if err == nil {
		t.Fatal("canceled rectification must fail")
	}
	current, findErr := repo.FindByID(id)
	if findErr != nil {
		t.Fatal(findErr)
	}
	if current.Status != constants.IncidentInvestigating {
		t.Fatalf("canceled update changed status to %s", current.Status)
	}
}

func formatID(v uint64) string {
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}
