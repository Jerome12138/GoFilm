package middleware

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

/*
Cors 跨域中间件.

设计:
  - 不再反射任意 Origin + Allow-Credentials=true (旧实现等同关闭同源策略).
  - 可信 Origin 通过 env CORS_ALLOWED_ORIGINS 注入, 逗号分隔.
  - env 未设置时默认放行 localhost / 127.0.0.1 + 容器内 service 名, 方便本地/开发.
  - Origin 不在白名单 → 不写任何 CORS 响应头, 浏览器自然拒绝, 后端代码仍正常处理请求
    (这意味着同源/服务端直连请求不会被这里挡住, 只挡跨域浏览器请求).
  - OPTIONS 一律 204, 不再回 "ok!" JSON.
  - 全局 panic recover 改写 500 (旧实现只 log 不响应, 客户端会拿到 ECONNRESET).
*/

var corsAllowedOrigins []string

func init() {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		// 本地/容器开发默认值. 生产必须通过 env 明确指定线上前端域名.
		corsAllowedOrigins = []string{
			"http://localhost",
			"http://localhost:80",
			"http://localhost:3600",
			"http://localhost:5173",
			"http://127.0.0.1",
			"http://127.0.0.1:3600",
			"http://127.0.0.1:5173",
		}
		log.Printf("WARN cors: CORS_ALLOWED_ORIGINS 未设置, 使用 dev 默认白名单. 生产环境必须显式设置.")
		return
	}
	for _, o := range strings.Split(raw, ",") {
		if v := strings.TrimSpace(o); v != "" {
			corsAllowedOrigins = append(corsAllowedOrigins, v)
		}
	}
}

func originAllowed(origin string) bool {
	for _, ok := range corsAllowedOrigins {
		if ok == "*" {
			return true
		}
		if ok == origin {
			return true
		}
	}
	return false
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		// panic 兜底: 写 500 让客户端拿到结构化错误而不是 ECONNRESET
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic recovered path=%s err=%v", c.Request.URL.Path, r)
				if !c.Writer.Written() {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
						"code": -1,
						"data": nil,
						"msg":  "服务器内部错误",
					})
				}
			}
		}()

		origin := c.Request.Header.Get("Origin")
		if origin != "" && originAllowed(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Content-Length, X-Requested-With")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type, new-token")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "172800")
			c.Header("Vary", "Origin")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
