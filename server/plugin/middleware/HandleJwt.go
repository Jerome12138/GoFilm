package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"server/config"
	"server/model/system"
)

// AuthToken 用户登录鉴权中间件.
//
// 协议:
//   - 请求头: Authorization: Bearer <token>
//   - 续期: 旧 token 过期且 redis 中和入参一致时, 中间件生成新 token,
//     通过响应头 new-token 下发 (原值, 不带 Bearer 前缀), 前端拦截器写回 store.
//
// 不再兼容历史的自定义 auth-token 请求头.
func AuthToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		authToken := extractBearer(c.Request.Header.Get("Authorization"))
		if authToken == "" {
			system.CustomResult(http.StatusUnauthorized, system.SUCCESS, nil, "用户未授权,请先登录", c)
			c.Abort()
			return
		}
		uc, err := system.ParseToken(authToken)
		// 非"过期"的解析错误一律拒绝, 同时防止 uc 为 nil 时后续解引用 panic
		if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
			system.CustomResult(http.StatusUnauthorized, system.SUCCESS, nil, "身份信息无效, 请重新登录", c)
			c.Abort()
			return
		}
		if uc == nil {
			system.CustomResult(http.StatusUnauthorized, system.SUCCESS, nil, "身份信息无效, 请重新登录", c)
			c.Abort()
			return
		}
		t := system.GetUserTokenById(uc.UserID)
		if len(t) <= 0 {
			system.CustomResult(http.StatusUnauthorized, system.SUCCESS, nil, "身份验证信息已失效,请重新登录!", c)
			c.Abort()
			return
		}
		if t != authToken {
			system.CustomResult(http.StatusUnauthorized, system.SUCCESS, nil, "账号在其它设备登录,身份验证信息失效,请重新登录!", c)
			c.Abort()
			return
		}
		if err != nil && errors.Is(err, jwt.ErrTokenExpired) {
			// token 与 redis 一致且过期 → 续发新 token
			newToken, _ := system.GenToken(uc.UserID, uc.UserName, uc.Role)
			_ = system.SaveUserToken(newToken, uc.UserID)
			uc, _ = system.ParseToken(newToken)
			c.Header("new-token", newToken)
		}
		c.Set(config.AuthUserClaims, uc)
		c.Next()
	}
}

// extractBearer 从 Authorization 头中提取 token, 仅接受 "Bearer <token>" 形式
// (前缀大小写不敏感). 缺失/格式错误返回空串, 由调用方按未鉴权处理.
func extractBearer(authz string) string {
	const prefix = "bearer"
	if len(authz) <= len(prefix) {
		return ""
	}
	if !strings.EqualFold(authz[:len(prefix)], prefix) {
		return ""
	}
	rest := authz[len(prefix):]
	// 必须有空白分隔, 且 token 非空
	if rest[0] != ' ' && rest[0] != '\t' {
		return ""
	}
	return strings.TrimSpace(rest)
}
