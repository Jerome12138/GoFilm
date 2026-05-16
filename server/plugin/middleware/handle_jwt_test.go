package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"server/config"
	"server/model/system"
	"server/plugin/db"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRedis(t *testing.T) func() {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	prev := db.Rdb
	db.Rdb = cli
	return func() {
		_ = cli.Close()
		mr.Close()
		db.Rdb = prev
	}
}

// dispatch 帮助调用 AuthToken 中间件 + downstream handler.
// token 非空则以 "Authorization: Bearer <token>" 形式注入.
func dispatch(token string) (int, http.Header) {
	r := gin.New()
	r.Use(AuthToken())
	r.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code, w.Header()
}

// dispatchRaw 直接设置 Authorization 头原文, 用于覆盖错误格式分支.
func dispatchRaw(authz string) (int, http.Header) {
	r := gin.New()
	r.Use(AuthToken())
	r.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code, w.Header()
}

func TestAuthToken_MissingHeaderReturns401(t *testing.T) {
	defer setupRedis(t)()
	code, _ := dispatch("")
	require.Equal(t, http.StatusUnauthorized, code)
}

func TestAuthToken_InvalidTokenReturns401_NoPanic(t *testing.T) {
	defer setupRedis(t)()
	code, _ := dispatch("garbage.token.value")
	require.Equal(t, http.StatusUnauthorized, code, "invalid token must not panic, must 401")
}

func TestAuthToken_ValidTokenWithoutRedisReturns401(t *testing.T) {
	defer setupRedis(t)()
	tok, err := system.GenToken(1, "alice", system.RoleNormal)
	require.NoError(t, err)
	// 故意不写入 redis → 视为登录已过期
	code, _ := dispatch(tok)
	require.Equal(t, http.StatusUnauthorized, code)
}

func TestAuthToken_ValidTokenAndRedisMismatchReturns401(t *testing.T) {
	defer setupRedis(t)()
	tok, err := system.GenToken(1, "alice", system.RoleNormal)
	require.NoError(t, err)
	// 写一个不同的 token, 模拟"账号在其它设备登录"
	require.NoError(t, system.SaveUserToken("other-token", 1))
	code, _ := dispatch(tok)
	require.Equal(t, http.StatusUnauthorized, code)
}

func TestAuthToken_ValidTokenAndRedisMatchPasses(t *testing.T) {
	defer setupRedis(t)()
	tok, err := system.GenToken(2, "bob", system.RoleNormal)
	require.NoError(t, err)
	require.NoError(t, system.SaveUserToken(tok, 2))
	code, _ := dispatch(tok)
	require.Equal(t, http.StatusNoContent, code, "valid token must reach downstream handler")
}

// 关键: 历史 bug 是无效 token 时 uc 为 nil 仍解引用 → panic.
// 这个 case 覆盖该修复.
func TestAuthToken_NilUserClaimsHandledGracefully(t *testing.T) {
	defer setupRedis(t)()
	// 一个签名但完全错误的 RSA token (用一个未知公钥)
	bad := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9." +
		"eyJzdWIiOiJ4In0." +
		"YQQ"
	code, _ := dispatch(bad)
	require.Equal(t, http.StatusUnauthorized, code, "nil claims must be handled, not panic")
	// 同时确保 config 包初始化加载了 RSA private key, 让 GenToken/ParseToken 实际工作
	_ = config.PrivateKey
}

// TestAuthToken_RejectsMissingBearerPrefix: 没有 Bearer 前缀直接 401, 不再兼容裸 token.
func TestAuthToken_RejectsMissingBearerPrefix(t *testing.T) {
	defer setupRedis(t)()
	tok, err := system.GenToken(1, "alice", system.RoleNormal)
	require.NoError(t, err)
	require.NoError(t, system.SaveUserToken(tok, 1))
	// 没加 Bearer 前缀, 应当被拒
	code, _ := dispatchRaw(tok)
	require.Equal(t, http.StatusUnauthorized, code)
}

// TestAuthToken_RejectsLegacyAuthTokenHeader: 旧版自定义 auth-token 头被弃用, 不再被识别.
func TestAuthToken_RejectsLegacyAuthTokenHeader(t *testing.T) {
	defer setupRedis(t)()
	tok, err := system.GenToken(1, "alice", system.RoleNormal)
	require.NoError(t, err)
	require.NoError(t, system.SaveUserToken(tok, 1))

	r := gin.New()
	r.Use(AuthToken())
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("auth-token", tok) // legacy header, 不再被读
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthToken_AcceptsLowercaseBearer: 前缀大小写不敏感.
func TestAuthToken_AcceptsLowercaseBearer(t *testing.T) {
	defer setupRedis(t)()
	tok, err := system.GenToken(7, "carol", system.RoleNormal)
	require.NoError(t, err)
	require.NoError(t, system.SaveUserToken(tok, 7))
	code, _ := dispatchRaw("bearer " + tok)
	require.Equal(t, http.StatusNoContent, code, "前缀大小写不应区分对待")
}

// TestExtractBearer 直接覆盖 extractBearer 的边界.
func TestExtractBearer(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"abc", ""},
		{"Bearer", ""},
		{"Bearer ", ""},                      // 空 token
		{"Bearer  abc", "abc"},               // 多空格也 trim
		{"Bearer\tabc", "abc"},               // tab 分隔
		{"Token xyz", ""},                    // 错误方案
		{"BearerXYZ", ""},                    // 没分隔
		{"Bearer foo.bar.baz", "foo.bar.baz"}, // 正常
	}
	for _, c := range cases {
		require.Equal(t, c.want, extractBearer(c.in), "input=%q", c.in)
	}
}
