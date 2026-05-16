package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// resetBuckets 清空 controller 内的内存频控状态, 单测之间隔离用.
func resetBuckets() {
	proxyBucketsMu.Lock()
	defer proxyBucketsMu.Unlock()
	proxyBuckets = map[string]*tokenBucket{}
}

// TestM3u8Proxy_MissingSrc: 缺 src 参数 → 业务码失败.
func TestM3u8Proxy_MissingSrc(t *testing.T) {
	resetBuckets()
	r := gin.New()
	r.GET("/m3u8/proxy", M3u8Proxy)
	req := httptest.NewRequest(http.MethodGet, "/m3u8/proxy", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "src")
}

// TestM3u8Proxy_RateLimit: 同 IP 高频请求触发 429.
// 默认 capacity=10, 第 11 次必应被拦.
func TestM3u8Proxy_RateLimit(t *testing.T) {
	resetBuckets()
	r := gin.New()
	r.GET("/m3u8/proxy", M3u8Proxy)
	// 前 10 次会真的去打 logic.FetchAndFilter, 但 src 解析失败前先消耗令牌; 这里给个肯定失败的 src,
	// 重点验 11th 是 429 (即先于业务校验直接拦).
	hit429 := false
	for i := 0; i < 11; i++ {
		req := httptest.NewRequest(http.MethodGet, "/m3u8/proxy?src=", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			hit429 = true
			break
		}
	}
	require.True(t, hit429, "突发 11 次同 IP 必被频控")
}

// TestM3u8Proxy_RateLimit_PerIP: 不同 IP 各自独立计数, A 满 B 不受影响.
func TestM3u8Proxy_RateLimit_PerIP(t *testing.T) {
	resetBuckets()
	r := gin.New()
	r.GET("/m3u8/proxy", M3u8Proxy)
	// 把 IP-A 打满
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/m3u8/proxy?src=", nil)
		req.RemoteAddr = "5.5.5.5:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
	// IP-B 第一次不应被拦
	req := httptest.NewRequest(http.MethodGet, "/m3u8/proxy?src=", nil)
	req.RemoteAddr = "6.6.6.6:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.NotEqual(t, http.StatusTooManyRequests, w.Code, "B IP 不应受 A IP 频控影响")
}
