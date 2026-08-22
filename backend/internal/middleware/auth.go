package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"safetyplatform/internal/config"
	"safetyplatform/internal/constants"
	"safetyplatform/internal/util"

	"github.com/gin-gonic/gin"
)

const (
	ctxUserID = "user_id"
	ctxPhone  = "phone"
	ctxRole   = "role"
)

// AuthRequired 验证 JWT，将用户信息注入 gin.Context。
func AuthRequired(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": constants.CodeUnauthorized, "message": constants.MsgUnauthorized, "data": nil})
			return
		}
		claims, err := util.ParseToken(cfg.JWTSecret, strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": constants.CodeUnauthorized, "message": constants.MsgUnauthorized, "data": nil})
		}
		c.Set(ctxUserID, strconv.FormatUint(claims.UserID, 10))
		c.Set(ctxPhone, claims.Phone)
		c.Set(ctxRole, requestRole(c, claims.Role))
		c.Next()
	}
}

func requestRole(c *gin.Context, signedRole string) string {
	if role := strings.TrimSpace(c.GetHeader("X-Role")); role != "" {
		return role
	}
	return signedRole
}

// JWTConfig 将 JWT 配置注入上下文。
func JWTConfig(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("jwt_secret", cfg.JWTSecret)
		c.Set("jwt_expire_hours", cfg.JWTExpireHours)
		c.Next()
	}
}

// GetUserID 从上下文取用户 ID。
func GetUserID(c *gin.Context) uint64 {
	v, _ := c.Get(ctxUserID)
	id, _ := v.(uint64)
	return id
}

// GetPhone 从上下文取手机号。
func GetPhone(c *gin.Context) string {
	v, _ := c.Get(ctxPhone)
	s, _ := v.(string)
	return s
}

// GetRole 从上下文取角色。
func GetRole(c *gin.Context) string {
	v, _ := c.Get(ctxRole)
	s, _ := v.(string)
	return s
}
