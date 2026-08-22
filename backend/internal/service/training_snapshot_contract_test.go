package service_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"safetyplatform/internal/handler"
	"safetyplatform/internal/model"
	"safetyplatform/internal/repository"
	"safetyplatform/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func trainingFixture(t *testing.T) (*repository.SafetyTrainingRepository, *service.SafetyTrainingService, uint64) {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.SafetyTraining{}); err != nil {
		t.Fatal(err)
	}
	training := &model.SafetyTraining{Topic: "tower crane", ParticipantIDs: model.JSONList{"u1", "u2", "u3"}}
	if err := db.Create(training).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.NewSafetyTrainingRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return repo, service.NewSafetyTrainingService(repo, repository.NewUserRepository(db), logger), training.ID
}

func TestTrainingSnapshotStableAfterRecord(t *testing.T) {
	_, svc, id := trainingFixture(t)
	before, err := svc.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Record(id, []string{"u7", "u8"}, 90); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(before.ParticipantIDs, ","); got != "u1,u2,u3" {
		t.Fatalf("old snapshot changed to %s", got)
	}
}

func TestTrainingFilterDoesNotMutateSource(t *testing.T) {
	source := model.JSONList{"crew-a", "guest", "crew-b"}
	filtered := model.FilterParticipantIDs(source, "crew-")
	if strings.Join(filtered, ",") != "crew-a,crew-b" {
		t.Fatalf("unexpected filter %v", filtered)
	}
	if strings.Join(source, ",") != "crew-a,guest,crew-b" {
		t.Fatalf("source changed to %v", source)
	}
}

func TestTrainingRepositoryReturnsIndependentSlice(t *testing.T) {
	repo, _, id := trainingFixture(t)
	first, err := repo.FindByID(id)
	if err != nil {
		t.Fatal(err)
	}
	first.ParticipantIDs[0] = "changed"
	second, err := repo.FindByID(id)
	if err != nil {
		t.Fatal(err)
	}
	if second.ParticipantIDs[0] != "u1" {
		t.Fatalf("repository exposed cached slice: %v", second.ParticipantIDs)
	}
}

func TestTrainingHandlerResponsesDoNotAlias(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo, svc, id := trainingFixture(t)
	exposed, err := repo.FindByID(id)
	if err != nil {
		t.Fatal(err)
	}
	exposed.ParticipantIDs[0] = "foreign"
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := handler.NewSafetyTrainingHandler(svc, logger)
	r := gin.New()
	r.GET("/trainings/:id", h.Get)
	req := httptest.NewRequest(http.MethodGet, "/trainings/"+trainingID(id), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var body struct {
		Data model.SafetyTraining `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.ParticipantIDs[0] != "u1" {
		t.Fatalf("handler returned contaminated slice: %s", rec.Body.String())
	}
}

func trainingID(v uint64) string {
	var b [20]byte
	i := len(b)
	for {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
		if v == 0 {
			return string(b[i:])
		}
	}
}
