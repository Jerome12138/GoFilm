package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"server/config"
	"server/model/system"
)

// RequireAdmin 角色限制中间件: 只允许 Role == RoleAdmin 的用户通过.
// 必须挂在 AuthToken 之后, 依赖其向 context 写入 *UserClaims.
//
// 401 与 403 区分:
//  - context 没有 claims → 401 (未登录或 token 无效, 由 AuthToken 拦截后此处理论不会触发,
//    保留兜底防止外部直接挂载本中间件)
//  - claims 存在但 Role 非管理员 → 403 (登录身份合法但权限不足)
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get(config.AuthUserClaims)
		if !ok {
			system.CustomResult(http.StatusUnauthorized, system.SUCCESS, nil, "用户未授权,请先登录", c)
			c.Abort()
			return
		}
		uc, ok := v.(*system.UserClaims)
		if !ok || uc == nil {
			system.CustomResult(http.StatusUnauthorized, system.SUCCESS, nil, "身份信息无效, 请重新登录", c)
			c.Abort()
			return
		}
		if uc.Role != system.RoleAdmin {
			system.CustomResult(http.StatusForbidden, system.FAILED, nil, "权限不足, 仅管理员可操作", c)
			c.Abort()
			return
		}
		c.Next()
	}
}
