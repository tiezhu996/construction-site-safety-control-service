package model_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"safetyplatform/internal/constants"
	"safetyplatform/internal/handler"
	"safetyplatform/internal/model"
	"safetyplatform/internal/repository"
	"safetyplatform/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func certReviewFixture(t *testing.T, photoURL string) (*gorm.DB, *repository.WorkerCertificationRepository, *service.WorkerCertificationService, *model.WorkerCertification) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.WorkerCertification{}); err != nil {
		t.Fatal(err)
	}
	cert := &model.WorkerCertification{UserID: 7, CertType: "electrician", CertNo: "E-7", CertPhotoURL: photoURL, Status: constants.CertPending}
	if err := db.Create(cert).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.NewWorkerCertificationRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.NewWorkerCertificationService(repo, repository.NewUserRepository(db), logger)
	return db, repo, svc, cert
}

func TestCertReviewNoNilMapPanic(t *testing.T) {
	_, _, svc, cert := certReviewFixture(t, "")
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("review panicked for a zero-value note map: %v", recovered)
		}
	}()
	got, err := svc.Review(cert.ID, constants.CertApproved)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReviewNotes["decision"] != constants.CertApproved {
		t.Fatalf("decision note = %q", got.ReviewNotes["decision"])
	}
}

func TestCertTypedNilPolicyRejected(t *testing.T) {
	policy := model.CertificationReviewPolicyFor(&model.WorkerCertification{})
	if policy != nil {
		t.Fatalf("empty evidence returned a non-nil policy of type %T", policy)
	}
}

func TestCertReviewFailureKeepsPending(t *testing.T) {
	db, repo, svc, cert := certReviewFixture(t, "ftp://internal/cert.jpg")
	if _, err := svc.Review(cert.ID, constants.CertApproved); err == nil {
		t.Fatal("invalid evidence was accepted")
	}
	var stored model.WorkerCertification
	if err := db.First(&stored, cert.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != constants.CertPending {
		t.Fatalf("failed review persisted status %q", stored.Status)
	}

	stale := stored
	if err := db.Model(&model.WorkerCertification{}).Where("id = ?", cert.ID).Update("status", constants.CertRejected).Error; err != nil {
		t.Fatal(err)
	}
	stale.Status = constants.CertApproved
	if err := repo.UpdateReview(&stale, constants.CertPending); err == nil {
		t.Fatal("stale pending review overwrote a terminal status")
	}
}

func TestCertReviewHandlerReportsValidation(t *testing.T) {
	_, _, svc, cert := certReviewFixture(t, "ftp://internal/cert.jpg")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := handler.NewWorkerCertificationHandler(svc, logger)
	body := bytes.NewBufferString(`{"status":"approved"}`)
	req := httptest.NewRequest(http.MethodPost, "/certifications/review", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = []gin.Param{{Key: "id", Value: strconv.FormatUint(cert.ID, 10)}}
	h.Review(c)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var response struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != constants.CodeValidationFailed {
		t.Fatalf("code = %d", response.Code)
	}
}
