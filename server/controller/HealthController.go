package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"server/plugin/db"
)

/*
健康检查端点.

  - GET /livez   → 进程存活, 始终 200 (gin 进入 handler 就算活)
  - GET /healthz → 依赖联通性, redis + mysql 任一 ping 失败返 503

docker compose / k8s 用 /healthz 做 readiness, /livez 做 liveness.
不挂鉴权中间件, 暴露给负载均衡和 health check 探针.
*/

// Livez 仅证明 HTTP handler 能进, 不查依赖.
func Livez(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Healthz 检查 redis + mysql 是否可用. 任一不可用返 503 + 详情, 让上游决定流量.
func Healthz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	checks := gin.H{}
	healthy := true

	if err := db.Rdb.Ping(ctx).Err(); err != nil {
		checks["redis"] = "fail: " + err.Error()
		healthy = false
	} else {
		checks["redis"] = "ok"
	}

	if sqlDB, err := db.Mdb.DB(); err != nil {
		checks["mysql"] = "fail: " + err.Error()
		healthy = false
	} else if err := sqlDB.PingContext(ctx); err != nil {
		checks["mysql"] = "fail: " + err.Error()
		healthy = false
	} else {
		checks["mysql"] = "ok"
	}

	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"status": map[bool]string{true: "ok", false: "fail"}[healthy], "checks": checks})
}
