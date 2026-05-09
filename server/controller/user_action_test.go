package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"server/config"
	"server/model/system"
	"server/plugin/db"
)

func withMockDB(t *testing.T) (sqlmock.Sqlmock, func()) {
	t.Helper()
	sqldb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	g, err := gorm.Open(mysql.New(mysql.Config{Conn: sqldb, SkipInitializeWithVersion: true}),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	prev := db.Mdb
	db.Mdb = g
	return mock, func() {
		db.Mdb = prev
		_ = sqldb.Close()
	}
}

// 注入"已登录"用户上下文, 给鉴权后路由直接使用
func withFakeAuth(uid uint, name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(config.AuthUserClaims, &system.UserClaims{UserID: uid, UserName: name})
		c.Next()
	}
}

// ============================== 注册 ==============================

func TestUserRegister_RejectsBadUsername(t *testing.T) {
	defer withMiniRedis(t)()
	mock, cleanup := withMockDB(t)
	defer cleanup()
	_ = mock // 期望不触发 SQL

	r := gin.New()
	r.POST("/user/register", UserRegister)

	body := []byte(`{"userName":"ab","password":"Abc1234!","email":"x@y.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "用户名格式")
	require.NoError(t, mock.ExpectationsWereMet(), "格式校验失败时不应触发 SQL")
}

func TestUserRegister_RejectsTakenUsername(t *testing.T) {
	defer withMiniRedis(t)()
	mock, cleanup := withMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`(?i)select count.+from .users. where user_name`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	r := gin.New()
	r.POST("/user/register", UserRegister)

	body := []byte(`{"userName":"alice123","password":"Abc1234!","email":"alice@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "用户名已被占用")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRegister_HappyPath(t *testing.T) {
	defer withMiniRedis(t)()
	mock, cleanup := withMockDB(t)
	defer cleanup()

	// 1. user_name 唯一性
	mock.ExpectQuery(`(?i)select count.+from .users. where user_name`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	// 2. email 唯一性
	mock.ExpectQuery(`(?i)select count.+from .users. where email`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	// 3. INSERT users
	mock.ExpectBegin()
	mock.ExpectExec(`(?i)insert into .users.`).WillReturnResult(sqlmock.NewResult(10001, 1))
	mock.ExpectCommit()

	r := gin.New()
	r.POST("/user/register", UserRegister)

	body := []byte(`{"userName":"alice123","password":"Abc1234!","email":"alice@example.com","nickName":"Alice"}`)
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
	require.Contains(t, got["msg"], "注册成功")
	require.NoError(t, mock.ExpectationsWereMet())
}

// ============================== 观看历史 ==============================

func TestHistoryUpsert_HappyPath(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()

	// GORM ON DUPLICATE KEY UPDATE 走单 INSERT
	mock.ExpectBegin()
	mock.ExpectExec(`(?i)insert into .user_histories.+on duplicate key update`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	r := gin.New()
	r.POST("/user/history", withFakeAuth(42, "alice"), HistoryUpsert)

	body := []byte(`{"mid":100,"cid":6,"name":"片名","picture":"p.png","playFrom":"siteA","episode":2,"progress":120}`)
	req := httptest.NewRequest(http.MethodPost, "/user/history", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHistoryUpsert_RejectsIncomplete(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()

	r := gin.New()
	r.POST("/user/history", withFakeAuth(42, "alice"), HistoryUpsert)

	body := []byte(`{"mid":0,"name":""}`)
	req := httptest.NewRequest(http.MethodPost, "/user/history", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "影片信息不完整")
	require.NoError(t, mock.ExpectationsWereMet(), "参数缺失时不应触发 SQL")
}

func TestHistoryDelete_RequiresIdentifier(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()

	r := gin.New()
	r.DELETE("/user/history", withFakeAuth(42, "alice"), HistoryDelete)

	req := httptest.NewRequest(http.MethodDelete, "/user/history", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "缺少 id 或 mid")
	require.NoError(t, mock.ExpectationsWereMet())
}

// ============================== 收藏 ==============================

func TestFavoriteAdd_Idempotent(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(`(?i)insert into .user_favorites.+on duplicate key update`).
		WillReturnResult(sqlmock.NewResult(1, 0))
	mock.ExpectCommit()

	r := gin.New()
	r.POST("/user/favorite", withFakeAuth(42, "alice"), FavoriteAdd)

	body := []byte(`{"mid":100,"cid":6,"name":"片名"}`)
	req := httptest.NewRequest(http.MethodPost, "/user/favorite", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFavoriteCheck_RequiresMid(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()

	r := gin.New()
	r.GET("/user/favorite/check", withFakeAuth(42, "alice"), FavoriteCheck)

	req := httptest.NewRequest(http.MethodGet, "/user/favorite/check", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "缺少 mid")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFavoriteCheck_ReturnsBoolean(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`(?i)select count.+from .user_favorites. where user_id`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	r := gin.New()
	r.GET("/user/favorite/check", withFakeAuth(42, "alice"), FavoriteCheck)

	req := httptest.NewRequest(http.MethodGet, "/user/favorite/check?mid=100", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
	data := got["data"].(map[string]interface{})
	require.Equal(t, true, data["favorited"])
	require.NoError(t, mock.ExpectationsWereMet())
}

// 鉴权失败: context 未注入用户信息
func TestUserAction_RequiresAuth(t *testing.T) {
	r := gin.New()
	r.POST("/user/favorite", FavoriteAdd) // 没挂 withFakeAuth

	req := httptest.NewRequest(http.MethodPost, "/user/favorite", bytes.NewReader([]byte(`{"mid":1,"name":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "登录")
}
