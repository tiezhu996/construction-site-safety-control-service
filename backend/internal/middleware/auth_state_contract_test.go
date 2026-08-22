package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"safetyplatform/internal/config"
	"safetyplatform/internal/constants"
	"safetyplatform/internal/middleware"
	"safetyplatform/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuthRejectsExpiredCredential(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "contract-secret"
	claims := util.Claims{
		UserID: 21,
		Phone:  "13800000021",
		Role:   constants.RoleWorker,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			Issuer:    "safety-platform",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	r := gin.New()
	r.Use(middleware.AuthRequired(&config.Config{JWTSecret: secret}))
	r.GET("/private", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expired credential status = %d", w.Code)
	}
}

func TestAuthMalformedCredentialDoesNotPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.AuthRequired(&config.Config{JWTSecret: "contract-secret"}))
	r.GET("/private", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer malformed.token.value")
	w := httptest.NewRecorder()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("malformed credential panicked: %v", recovered)
		}
	}()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("malformed credential status = %d", w.Code)
	}
}

func TestAuthIgnoresUnsignedRoleHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "contract-secret"
	token, err := util.GenerateToken(secret, time.Hour, 31, "13800000031", constants.RoleWorker)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	r := gin.New()
	r.Use(middleware.AuthRequired(&config.Config{JWTSecret: secret}), middleware.RequireRole(constants.RoleAdmin))
	r.GET("/admin", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Role", constants.RoleAdmin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("unsigned role header status = %d", w.Code)
	}
}

func TestAuthPreservesNumericUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "contract-secret"
	token, err := util.GenerateToken(secret, time.Hour, 47, "13800000047", constants.RoleInspector)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	r := gin.New()
	r.Use(middleware.AuthRequired(&config.Config{JWTSecret: secret}))
	r.GET("/identity", func(c *gin.Context) {
		if middleware.GetUserID(c) != 47 {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/identity", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("identity status = %d", w.Code)
	}
}

func TestRoleRequiresExactSignedValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "contract-secret"
	token, err := util.GenerateToken(secret, time.Hour, 59, "13800000059", "superadmin")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	r := gin.New()
	r.Use(middleware.AuthRequired(&config.Config{JWTSecret: secret}), middleware.RequireRole(constants.RoleAdmin))
	r.GET("/admin", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("role prefix status = %d", w.Code)
	}
}
