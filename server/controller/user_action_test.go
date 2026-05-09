package controller

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
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

// encryptPwdForTest 复刻 util.PasswordEncrypt 的 (pwd+salt) md5*3 算法,
// 仅用于测试构造 mock 期望值, 不依赖实现细节.
func encryptPwdForTest(password, salt string) string {
	b := []byte(password + salt)
	var r [16]byte
	for i := 0; i < 3; i++ {
		r = md5.Sum(b)
		b = []byte(hex.EncodeToString(r[:]))
	}
	return hex.EncodeToString(r[:])
}

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

// ============================== 管理员创建用户 ==============================

func TestManageUserCreate_RejectsBadUsername(t *testing.T) {
	defer withMiniRedis(t)()
	mock, cleanup := withMockDB(t)
	defer cleanup()
	_ = mock // 期望不触发 SQL

	r := gin.New()
	r.POST("/manage/user/create", ManageUserCreate)

	body := []byte(`{"userName":"ab","password":"Abc1234!","email":"x@y.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/manage/user/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "用户名格式")
	require.NoError(t, mock.ExpectationsWereMet(), "格式校验失败时不应触发 SQL")
}

func TestManageUserCreate_RejectsTakenUsername(t *testing.T) {
	defer withMiniRedis(t)()
	mock, cleanup := withMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`(?i)select count.+from .users. where user_name`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	r := gin.New()
	r.POST("/manage/user/create", ManageUserCreate)

	body := []byte(`{"userName":"alice123","password":"Abc1234!","email":"alice@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/manage/user/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "用户名已被占用")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestManageUserCreate_HappyPath(t *testing.T) {
	defer withMiniRedis(t)()
	mock, cleanup := withMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`(?i)select count.+from .users. where user_name`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`(?i)select count.+from .users. where email`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectExec(`(?i)insert into .users.`).WillReturnResult(sqlmock.NewResult(10001, 1))
	mock.ExpectCommit()

	r := gin.New()
	r.POST("/manage/user/create", ManageUserCreate)

	body := []byte(`{"userName":"alice123","password":"Abc1234!","email":"alice@example.com","nickName":"Alice","role":0}`)
	req := httptest.NewRequest(http.MethodPost, "/manage/user/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
	require.Contains(t, got["msg"], "用户创建成功")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestManageUserCreate_RejectsBadRole(t *testing.T) {
	defer withMiniRedis(t)()
	mock, cleanup := withMockDB(t)
	defer cleanup()

	r := gin.New()
	r.POST("/manage/user/create", ManageUserCreate)

	body := []byte(`{"userName":"alice123","password":"Abc1234!","role":9}`)
	req := httptest.NewRequest(http.MethodPost, "/manage/user/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "角色取值非法")
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

// ============================== 用户自助修改密码 ==============================

// 普通用户走自己的 token 修改自己密码: 旧密码校验通过 → UPDATE users
func TestUserPasswordChange_HappyPath(t *testing.T) {
	defer withMiniRedis(t)()
	mock, cleanup := withMockDB(t)
	defer cleanup()

	// 旧密码加密前: "OldPwd1!" + salt → md5*3 后存表; 这里直接 mock SELECT 返回任意密码 hash
	// 实际校验逻辑: util.PasswordEncrypt("OldPwd1!", salt) == 数据库里的 password
	// 因此我们让 SELECT 返回 (password=PasswordEncrypt("OldPwd1!", salt), salt)
	salt := "DEADBEEF"
	oldHash := encryptPwdForTest("OldPwd1!", salt)

	// GetUserByNameOrEmail: SELECT * FROM users WHERE (user_name = ? OR email = ?) ...
	mock.ExpectQuery(`(?i)select .+ from .users. where .*user_name`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_name", "password", "salt"}).
			AddRow(uint(42), "alice", oldHash, salt))
	mock.ExpectBegin()
	// UpdateUserInfo: UPDATE users SET password = ?
	mock.ExpectExec(`(?i)update .users. set`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	r := gin.New()
	r.POST("/user/changePassword", withFakeAuth(42, "alice"), UserPasswordChange)

	body := []byte(`{"password":"OldPwd1!","newPassword":"NewPwd1!"}`)
	req := httptest.NewRequest(http.MethodPost, "/user/changePassword", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
	require.Contains(t, got["msg"], "密码修改成功")
	require.NoError(t, mock.ExpectationsWereMet())
}

// 新密码不合规 (没特殊字符) → 不应触发任何 SQL
func TestUserPasswordChange_RejectsWeakNewPassword(t *testing.T) {
	defer withMiniRedis(t)()
	mock, cleanup := withMockDB(t)
	defer cleanup()

	r := gin.New()
	r.POST("/user/changePassword", withFakeAuth(42, "alice"), UserPasswordChange)

	body := []byte(`{"password":"OldPwd1!","newPassword":"weakpass"}`)
	req := httptest.NewRequest(http.MethodPost, "/user/changePassword", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "密码格式校验失败")
	require.NoError(t, mock.ExpectationsWereMet(), "新密码不合规时不应触发 SQL")
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
