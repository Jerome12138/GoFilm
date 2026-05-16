package controller

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"server/logic"
	"server/model/system"
)

/*
M3u8Proxy: 拉取上游 m3u8, 过滤疑似广告 segment 后以 application/vnd.apple.mpegurl 直接吐回.

防御:
 1. SSRF: logic.M3UL 内部做 DNS + dial 双层 IP 校验, 详见 M3u8Logic.go 顶部注释.
 2. 错误脱敏: 上游具体错误日志写到 server side, 客户端只看到通用提示,
    避免变成内网扫描器 (通过观察 "connection refused" / "host not found" 等串).
 3. 频控: per-IP token bucket, 默认 10 req / 60s. 滥用即 429.
*/

// proxyBuckets 简易内存频控: ip → tokenBucket. 单进程内有效, 多副本部署需上游代理层兜底.
type tokenBucket struct {
	tokens    float64
	lastFill  time.Time
	mu        sync.Mutex
	capacity  float64
	refillSec float64 // 每秒补几个 token
}

const (
	m3u8RateCapacity = 10.0       // bucket 容量
	m3u8RateRefill   = 10.0 / 60. // 每秒补 1/6 (即 10 req/60s 稳态)
)

var (
	proxyBuckets   = map[string]*tokenBucket{}
	proxyBucketsMu sync.Mutex
)

// allow 返回 true 表示放行, false 表示已超额. 同时清理一段时间不再活跃的 bucket.
func allow(ip string) bool {
	proxyBucketsMu.Lock()
	b := proxyBuckets[ip]
	if b == nil {
		b = &tokenBucket{
			tokens:    m3u8RateCapacity,
			lastFill:  time.Now(),
			capacity:  m3u8RateCapacity,
			refillSec: m3u8RateRefill,
		}
		proxyBuckets[ip] = b
	}
	proxyBucketsMu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(b.lastFill).Seconds()
	b.tokens += elapsed * b.refillSec
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.lastFill = now
	if b.tokens < 1 {
		return false
	}
	b.tokens -= 1
	return true
}

// M3u8Proxy GET /m3u8/proxy?src=<urlencoded>
func M3u8Proxy(c *gin.Context) {
	ip := c.ClientIP()
	if !allow(ip) {
		system.CustomResult(http.StatusTooManyRequests, system.FAILED, nil, "请求过于频繁, 请稍候再试", c)
		return
	}

	src := strings.TrimSpace(c.Query("src"))
	if src == "" {
		system.Failed("缺少 src 参数", c)
		return
	}

	filtered, err := logic.M3UL.FetchAndFilter(src)
	if err != nil {
		// 内部细节 (具体上游 host / 错误码 / DNS 错误) 只打日志, 不回客户端
		log.Printf("m3u8 proxy err src=%q ip=%s err=%v", src, ip, err)
		switch {
		case errors.Is(err, logic.ErrBlockedHost):
			system.Failed("不允许的目标地址", c)
		default:
			system.Failed("m3u8 拉取失败", c)
		}
		return
	}
	// 用户级私有缓存 (CDN/反代不缓存), 配合 src 不同每个用户/影片各自缓存
	c.Header("Cache-Control", "private, max-age=600")
	c.Data(http.StatusOK, "application/vnd.apple.mpegurl; charset=utf-8", []byte(filtered))
}
