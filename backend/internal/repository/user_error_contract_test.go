package repository_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"safetyplatform/internal/constants"
	"safetyplatform/internal/handler"
	"safetyplatform/internal/model"
	"safetyplatform/internal/repository"
	"safetyplatform/internal/service"
	"safetyplatform/internal/util"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openUserDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func userLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestUserMissingErrorChain(t *testing.T) {
	repo := repository.NewUserRepository(openUserDB(t))
	_, err := repo.FindByPhone("13900009999")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("missing phone must preserve ErrNotFound, got %v", err)
	}
}

func TestUserDuplicateRegistrationConflict(t *testing.T) {
	repo := repository.NewUserRepository(openUserDB(t))
	svc := service.NewUserService(repo, userLogger())
	if _, err := svc.Register("13900000001", "secret1", "first", "worker"); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}
	_, err := svc.Register("13900000001", "secret1", "second", "worker")
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Code != constants.CodeUserExists {
		t.Fatalf("duplicate must remain a classified conflict, got %v", err)
	}
}

func TestUserInvalidLoginUnauthorized(t *testing.T) {
	repo := repository.NewUserRepository(openUserDB(t))
	svc := service.NewUserService(repo, userLogger())
	_, _, err := svc.Login("secret", 1, "13900008888", "missing")
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Code != constants.CodeInvalidCredentials {
		t.Fatalf("missing login must be unauthorized, got %v", err)
	}
}

func TestUserLookupNotFoundResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewUserRepository(openUserDB(t))
	h := handler.NewUserHandler(service.NewUserService(repo, userLogger()), userLogger())
	r := gin.New()
	r.GET("/me", func(c *gin.Context) {
		c.Set("user_id", uint64(9999))
		h.Me(c)
	})
	req := httptest.NewRequest(http.MethodGet, "/me", bytes.NewReader(nil))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusNotFound || body.Code != constants.CodeNotFound {
		t.Fatalf("missing profile must be 404/code %d, got status=%d body=%s", constants.CodeNotFound, rec.Code, rec.Body.String())
	}
}
