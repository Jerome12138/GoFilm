package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"server/config"
	"server/model/system"
)

// 给定一个 RoleN 的 claims, 跑一次 RequireAdmin 中间件, 返回响应状态码.
func runRequireAdmin(role int, injectClaims bool) int {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if injectClaims {
			c.Set(config.AuthUserClaims, &system.UserClaims{UserID: 1, UserName: "u", Role: role})
		}
		c.Next()
	})
	r.Use(RequireAdmin())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestRequireAdmin_AllowsAdmin(t *testing.T) {
	require.Equal(t, http.StatusNoContent, runRequireAdmin(system.RoleAdmin, true))
}

func TestRequireAdmin_RejectsNormalUserWith403(t *testing.T) {
	require.Equal(t, http.StatusForbidden, runRequireAdmin(system.RoleNormal, true))
}

func TestRequireAdmin_NoClaimsReturns401(t *testing.T) {
	require.Equal(t, http.StatusUnauthorized, runRequireAdmin(0, false))
}
