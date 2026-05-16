package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"server/config"
	"server/model/system"
	"server/plugin/db"
	"server/plugin/common/util"
)

// =============================== /login (POST) ===============================

func TestLogin_RejectsBadJSON(t *testing.T) {
	r := gin.New()
	r.POST("/login", Login)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "登录信息异常")
}

func TestLogin_RejectsEmptyCredentials(t *testing.T) {
	r := gin.New()
	r.POST("/login", Login)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte(`{"userName":"","password":""}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "不能为空")
}

func TestLogin_HappyPathReturnsStructuredBody(t *testing.T) {
	defer withMiniRedis(t)()
	mock, cleanup := withMockDB(t)
	defer cleanup()

	salt := "ABCD1234"
	pwdHash := encryptPwdForTest("Abc1234!", salt)
	mock.ExpectQuery(`(?i)select .+ from .users. where .*user_name`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_name", "password", "salt", "role"}).
			AddRow(uint(42), "alice", pwdHash, salt, system.RoleAdmin))

	r := gin.New()
	r.POST("/login", Login)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte(`{"userName":"alice","password":"Abc1234!"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
	require.Empty(t, w.Header().Get("new-token"), "登录路径不再用 new-token 头, 改放 body")

	data, ok := got["data"].(map[string]interface{})
	require.True(t, ok, "data 必须是对象, 含 token/expires/role/userName")
	require.Equal(t, "alice", data["userName"])
	require.NotEmpty(t, data["token"], "登录成功 body 必须返回 token")
	require.Greater(t, data["expires"], float64(0), "expires 必须是大于 0 的 unix 秒")
	require.EqualValues(t, system.RoleAdmin, data["role"])
}

func TestLogin_RejectsWrongPassword(t *testing.T) {
	defer withMiniRedis(t)()
	mock, cleanup := withMockDB(t)
	defer cleanup()

	salt := "WRONG"
	pwdHash := encryptPwdForTest("RealPwd1!", salt)
	mock.ExpectQuery(`(?i)select .+ from .users. where .*user_name`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_name", "password", "salt", "role"}).
			AddRow(uint(1), "alice", pwdHash, salt, 0))

	r := gin.New()
	r.POST("/login", Login)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte(`{"userName":"alice","password":"WrongPwd1!"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "用户名或密码错误")
}

// =============================== /logout (GET, 鉴权) ===============================

func TestLogout_HappyPath(t *testing.T) {
	defer withMiniRedis(t)()
	r := gin.New()
	r.GET("/logout", withFakeAuth(42, "alice"), Logout)

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
	require.Contains(t, got["msg"], "已退出")
}

func TestLogout_NoAuthClaimsReturnsFailure(t *testing.T) {
	defer withMiniRedis(t)()
	r := gin.New()
	r.GET("/logout", Logout) // 没挂 withFakeAuth

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "登录")
}

// =============================== /user/info (GET, 鉴权) ===============================

func TestUserInfo_HappyPath(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()
	mock.ExpectQuery(`(?i)select .+ from .users.+id`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_name", "email", "nick_name", "role"}).
			AddRow(uint(42), "alice", "a@x.com", "Alice", system.RoleNormal))

	r := gin.New()
	r.GET("/user/info", withFakeAuth(42, "alice"), UserInfo)

	req := httptest.NewRequest(http.MethodGet, "/user/info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
	data := got["data"].(map[string]interface{})
	require.Equal(t, "alice", data["userName"])
	require.EqualValues(t, system.RoleNormal, data["role"])
}

func TestUserInfo_NoAuthClaimsReturnsFailure(t *testing.T) {
	r := gin.New()
	r.GET("/user/info", UserInfo) // 没挂 withFakeAuth

	req := httptest.NewRequest(http.MethodGet, "/user/info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "授权")
}

// =============================== /config/basic (GET) ===============================

func TestSiteBasicConfig_HappyPath(t *testing.T) {
	defer withMiniRedis(t)()
	bc := system.BasicConfig{SiteName: "GoFilm", Domain: "http://127.0.0.1:3601"}
	raw, _ := json.Marshal(bc)
	require.NoError(t, db.Rdb.Set(db.Cxt, config.SiteConfigBasic, raw, 0).Err())

	r := gin.New()
	r.GET("/config/basic", SiteBasicConfig)
	req := httptest.NewRequest(http.MethodGet, "/config/basic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
	data := got["data"].(map[string]interface{})
	require.Equal(t, "GoFilm", data["siteName"])
}

func TestSiteBasicConfig_RedisMissReturnsZeroValue(t *testing.T) {
	defer withMiniRedis(t)()
	r := gin.New()
	r.GET("/config/basic", SiteBasicConfig)
	req := httptest.NewRequest(http.MethodGet, "/config/basic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"], "redis 没数据时仍 200, 返回零值结构")
}

// =============================== /filmPlayInfo (GET) ===============================

func TestFilmPlayInfo_BadIdReturnsFailure(t *testing.T) {
	r := gin.New()
	r.GET("/filmPlayInfo", FilmPlayInfo)
	req := httptest.NewRequest(http.MethodGet, "/filmPlayInfo?id=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "影片ID")
}

func TestFilmPlayInfo_BadEpisodeReturnsFailure(t *testing.T) {
	r := gin.New()
	r.GET("/filmPlayInfo", FilmPlayInfo)
	req := httptest.NewRequest(http.MethodGet, "/filmPlayInfo?id=1&episode=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "集数")
}

// =============================== /searchFilm (GET) ===============================

// 关键字未命中: page.Total <= 0 → 业务失败 + "暂无相关影片信息"
func TestSearchFilm_NoMatchReturnsFailure(t *testing.T) {
	defer withMiniRedis(t)()
	mock, cleanup := withMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`(?i)select count.+from .search.`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`(?i)select .+ from .search.`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "mid"}))

	r := gin.New()
	r.GET("/searchFilm", SearchFilm)
	req := httptest.NewRequest(http.MethodGet, "/searchFilm?keyword=xxx", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "暂无")
}

// =============================== 让 util 引用不浪费 import ===============================
var _ = util.GenerateSalt
