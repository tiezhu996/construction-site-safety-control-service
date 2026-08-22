package middleware

import (
	"net/http"
	"strings"

	"safetyplatform/internal/constants"

	"github.com/gin-gonic/gin"
)

// RequireRole 基于用户角色校验权限。
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := GetRole(c)
		for _, r := range roles {
			if strings.Contains(role, r) {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": constants.CodeForbidden, "message": constants.MsgForbidden, "data": nil})
	}
}
