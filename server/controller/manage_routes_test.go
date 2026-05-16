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
)

// 这一组测试是 manage 路由的 controller 单元测试.
// 已通过 plugin/middleware/require_admin_test.go 单独验证 RequireAdmin 行为,
// 因此这里直接调用 controller, 不再每个用例重复挂中间件.

// ===================== /manage/index =====================
// GetDashboardStat 内会调 GetCollectSourceList (redis) / CountFilms (mysql) /
// GetAllFilmTask (redis), 任一空依赖都会 nil 解引用 panic.
// 这里同时挂 mini redis + sqlmock + count 预期, 测 happy path.
func TestManageIndex_HappyPath(t *testing.T) {
	defer withMiniRedis(t)()
	mock, cleanup := withMockDB(t)
	defer cleanup()
	// CountFilms() → SELECT count(*) FROM search
	mock.ExpectQuery(`(?i)select count.+from .search.`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	r := gin.New()
	r.GET("/manage/index", ManageIndex)
	req := httptest.NewRequest(http.MethodGet, "/manage/index", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
	data, ok := got["data"].(map[string]interface{})
	require.True(t, ok)
	require.Contains(t, data, "filmCount")
	require.Contains(t, data, "collectCount")
	require.Contains(t, data, "cronCount")
}

// ===================== /manage/user/list =====================
func TestManageUserList_HappyPath(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`(?i)select count.+from .users.`).WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`(?i)select .+ from .users.`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "user_name", "email", "nick_name", "role"}).
			AddRow(uint(10000), "admin", "a@x", "Z", system.RoleAdmin).
			AddRow(uint(10001), "alice", "b@x", "A", system.RoleNormal))

	r := gin.New()
	r.GET("/manage/user/list", ManageUserList)
	req := httptest.NewRequest(http.MethodGet, "/manage/user/list?current=1&pageSize=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
	data := got["data"].(map[string]interface{})
	list := data["list"].([]interface{})
	require.Len(t, list, 2)
	require.NoError(t, mock.ExpectationsWereMet())
}

// ===================== /manage/config/* =====================

func TestUpdateSiteBasic_RejectsBadJSON(t *testing.T) {
	r := gin.New()
	r.POST("/manage/config/basic/update", UpdateSiteBasic)
	req := httptest.NewRequest(http.MethodPost, "/manage/config/basic/update", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestUpdateSiteBasic_RejectsBadDomain(t *testing.T) {
	r := gin.New()
	r.POST("/manage/config/basic/update", UpdateSiteBasic)
	body := []byte(`{"siteName":"X","domain":"not-a-domain"}`)
	req := httptest.NewRequest(http.MethodPost, "/manage/config/basic/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "域名")
}

func TestUpdateSiteBasic_HappyPath(t *testing.T) {
	defer withMiniRedis(t)()
	r := gin.New()
	r.POST("/manage/config/basic/update", UpdateSiteBasic)
	body := []byte(`{"siteName":"GoFilm","domain":"http://127.0.0.1:3601"}`)
	req := httptest.NewRequest(http.MethodPost, "/manage/config/basic/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
}

func TestResetSiteBasic_HappyPath(t *testing.T) {
	defer withMiniRedis(t)()
	r := gin.New()
	r.GET("/manage/config/basic/reset", ResetSiteBasic)
	req := httptest.NewRequest(http.MethodGet, "/manage/config/basic/reset", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
}

// ===================== /manage/collect/* =====================

func TestFilmSourceList_EmptyRedis(t *testing.T) {
	defer withMiniRedis(t)()
	r := gin.New()
	r.GET("/manage/collect/list", FilmSourceList)
	req := httptest.NewRequest(http.MethodGet, "/manage/collect/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"], "空列表也应返回 0/成功")
}

func TestFilmSourceList_HappyPath(t *testing.T) {
	defer withMiniRedis(t)()
	// 写一条采集站到 redis
	fs := system.FilmSource{Id: "siteA", Name: "测试站", Uri: "http://x", Grade: system.MasterCollect, ResultModel: system.JsonResult, CollectType: system.CollectVideo, State: true}
	require.NoError(t, system.SaveCollectSourceList([]system.FilmSource{fs}))

	r := gin.New()
	r.GET("/manage/collect/list", FilmSourceList)
	req := httptest.NewRequest(http.MethodGet, "/manage/collect/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
	list := got["data"].([]interface{})
	require.Len(t, list, 1)
}

func TestFindFilmSource_RequiresId(t *testing.T) {
	r := gin.New()
	r.GET("/manage/collect/find", FindFilmSource)
	req := httptest.NewRequest(http.MethodGet, "/manage/collect/find", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "标识")
}

func TestFindFilmSource_NotFound(t *testing.T) {
	defer withMiniRedis(t)()
	r := gin.New()
	r.GET("/manage/collect/find", FindFilmSource)
	req := httptest.NewRequest(http.MethodGet, "/manage/collect/find?id=ghost", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "不存在")
}

func TestFilmSourceAdd_RejectsBadJSON(t *testing.T) {
	r := gin.New()
	r.POST("/manage/collect/add", FilmSourceAdd)
	req := httptest.NewRequest(http.MethodPost, "/manage/collect/add", bytes.NewReader([]byte("x")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestFilmSourceAdd_RejectsInvalidUri(t *testing.T) {
	r := gin.New()
	r.POST("/manage/collect/add", FilmSourceAdd)
	body := []byte(`{"name":"x","uri":"not-url","resultModel":0,"collectType":0}`)
	req := httptest.NewRequest(http.MethodPost, "/manage/collect/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "链接")
}

func TestFilmSourceUpdate_RejectsBadJSON(t *testing.T) {
	r := gin.New()
	r.POST("/manage/collect/update", FilmSourceUpdate)
	req := httptest.NewRequest(http.MethodPost, "/manage/collect/update", bytes.NewReader([]byte("x")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestFilmSourceChange_RejectsBadJSON(t *testing.T) {
	r := gin.New()
	r.POST("/manage/collect/change", FilmSourceChange)
	req := httptest.NewRequest(http.MethodPost, "/manage/collect/change", bytes.NewReader([]byte("x")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestFilmSourceChange_RequiresId(t *testing.T) {
	r := gin.New()
	r.POST("/manage/collect/change", FilmSourceChange)
	body := []byte(`{"id":"","state":true}`)
	req := httptest.NewRequest(http.MethodPost, "/manage/collect/change", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "标识")
}

func TestFilmSourceDel_RequiresId(t *testing.T) {
	r := gin.New()
	r.GET("/manage/collect/del", FilmSourceDel)
	req := httptest.NewRequest(http.MethodGet, "/manage/collect/del", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestFilmSourceDel_NotFound(t *testing.T) {
	defer withMiniRedis(t)()
	r := gin.New()
	r.GET("/manage/collect/del", FilmSourceDel)
	req := httptest.NewRequest(http.MethodGet, "/manage/collect/del?id=ghost", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestFilmSourceTest_RejectsBadJSON(t *testing.T) {
	r := gin.New()
	r.POST("/manage/collect/test", FilmSourceTest)
	req := httptest.NewRequest(http.MethodPost, "/manage/collect/test", bytes.NewReader([]byte("x")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestGetNormalFilmSource_HappyPath(t *testing.T) {
	defer withMiniRedis(t)()
	fs := system.FilmSource{Id: "siteA", Name: "X", Uri: "http://x", State: true, Grade: system.SlaveCollect, ResultModel: system.JsonResult, CollectType: system.CollectVideo}
	require.NoError(t, system.SaveCollectSourceList([]system.FilmSource{fs}))

	r := gin.New()
	r.GET("/manage/collect/options", GetNormalFilmSource)
	req := httptest.NewRequest(http.MethodGet, "/manage/collect/options", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
	list := got["data"].([]interface{})
	require.Len(t, list, 1)
}

// ===================== /manage/cron/* =====================

func TestFilmCronTaskList_EmptyReturnsFailure(t *testing.T) {
	defer withMiniRedis(t)()
	r := gin.New()
	r.GET("/manage/cron/list", FilmCronTaskList)
	req := httptest.NewRequest(http.MethodGet, "/manage/cron/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "暂无")
}

func TestGetFilmCronTask_RequiresId(t *testing.T) {
	r := gin.New()
	r.GET("/manage/cron/find", GetFilmCronTask)
	req := httptest.NewRequest(http.MethodGet, "/manage/cron/find", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestGetFilmCronTask_NotFound(t *testing.T) {
	defer withMiniRedis(t)()
	r := gin.New()
	r.GET("/manage/cron/find", GetFilmCronTask)
	req := httptest.NewRequest(http.MethodGet, "/manage/cron/find?id=ghost", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestFilmCronAdd_RejectsBadJSON(t *testing.T) {
	r := gin.New()
	r.POST("/manage/cron/add", FilmCronAdd)
	req := httptest.NewRequest(http.MethodPost, "/manage/cron/add", bytes.NewReader([]byte("x")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestFilmCronAdd_RejectsBadModel(t *testing.T) {
	r := gin.New()
	r.POST("/manage/cron/add", FilmCronAdd)
	body := []byte(`{"model":99,"time":3,"spec":"0 */20 * * * ?"}`)
	req := httptest.NewRequest(http.MethodPost, "/manage/cron/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "任务类型")
}

func TestFilmCronAdd_RejectsZeroTime(t *testing.T) {
	r := gin.New()
	r.POST("/manage/cron/add", FilmCronAdd)
	body := []byte(`{"model":0,"time":0,"spec":"0 */20 * * * ?"}`)
	req := httptest.NewRequest(http.MethodPost, "/manage/cron/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "采集时长")
}

func TestFilmCronUpdate_RejectsBadJSON(t *testing.T) {
	r := gin.New()
	r.POST("/manage/cron/update", FilmCronUpdate)
	req := httptest.NewRequest(http.MethodPost, "/manage/cron/update", bytes.NewReader([]byte("x")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestChangeTaskState_RejectsBadJSON(t *testing.T) {
	r := gin.New()
	r.POST("/manage/cron/change", ChangeTaskState)
	req := httptest.NewRequest(http.MethodPost, "/manage/cron/change", bytes.NewReader([]byte("x")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestDelFilmCron_RequiresId(t *testing.T) {
	r := gin.New()
	r.GET("/manage/cron/del", DelFilmCron)
	req := httptest.NewRequest(http.MethodGet, "/manage/cron/del", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

// ===================== /manage/spider/* =====================

func TestStarSpider_RejectsBadJSON(t *testing.T) {
	r := gin.New()
	r.POST("/manage/spider/start", StarSpider)
	req := httptest.NewRequest(http.MethodPost, "/manage/spider/start", bytes.NewReader([]byte("x")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestStarSpider_RejectsZeroTime(t *testing.T) {
	r := gin.New()
	r.POST("/manage/spider/start", StarSpider)
	body := []byte(`{"id":"x","time":0}`)
	req := httptest.NewRequest(http.MethodPost, "/manage/spider/start", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "采集时长")
}

func TestStarSpider_BatchEmptyIdsRejected(t *testing.T) {
	r := gin.New()
	r.POST("/manage/spider/start", StarSpider)
	body := []byte(`{"batch":true,"time":3,"ids":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/manage/spider/start", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestSpiderReset_RejectsWrongKey(t *testing.T) {
	r := gin.New()
	r.GET("/manage/spider/zero", SpiderReset)
	req := httptest.NewRequest(http.MethodGet, "/manage/spider/zero?accessKey=wrong", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "密钥")
}

func TestCoverFilmClass_NoMasterReturnsFailure(t *testing.T) {
	defer withMiniRedis(t)()
	r := gin.New()
	r.GET("/manage/spider/class/cover", CoverFilmClass)
	req := httptest.NewRequest(http.MethodGet, "/manage/spider/class/cover", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

// ===================== /manage/film/* =====================

func TestFilmAdd_RejectsBadJSON(t *testing.T) {
	r := gin.New()
	r.POST("/manage/film/add", FilmAdd)
	req := httptest.NewRequest(http.MethodPost, "/manage/film/add", bytes.NewReader([]byte("x")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

// FilmAdd 的 happy path 涉及 ConvertFilmDetailVo + SaveDetail (mysql + redis 写),
// 单元层 mock 代价高且与 model 层重复; 这里仅覆盖参数解析失败路径.
// 完整链路靠 model/system 层的 SaveDetail 测试 + 集成测试覆盖.
func TestFilmAdd_AcceptsRequestStructure(t *testing.T) {
	t.Skip("happy path 依赖 mysql + redis 全链路 mock, 由 model 层与集成测试覆盖")
}

func TestFilmSearchPage_RejectsBadPid(t *testing.T) {
	r := gin.New()
	r.GET("/manage/film/search/list", FilmSearchPage)
	req := httptest.NewRequest(http.MethodGet, "/manage/film/search/list?pid=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestFilmClassTree_HappyPath(t *testing.T) {
	defer withMiniRedis(t)()
	tree := system.CategoryTree{Category: &system.Category{Id: 0, Name: "root"}}
	raw, _ := json.Marshal(tree)
	require.NoError(t, db.Rdb.Set(db.Cxt, config.CategoryTreeKey, raw, 0).Err())

	r := gin.New()
	r.GET("/manage/film/class/tree", FilmClassTree)
	req := httptest.NewRequest(http.MethodGet, "/manage/film/class/tree", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
}

func TestFindFilmClass_RequiresId(t *testing.T) {
	r := gin.New()
	r.GET("/manage/film/class/find", FindFilmClass)
	req := httptest.NewRequest(http.MethodGet, "/manage/film/class/find", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestFindFilmClass_NotFound(t *testing.T) {
	defer withMiniRedis(t)()
	r := gin.New()
	r.GET("/manage/film/class/find", FindFilmClass)
	req := httptest.NewRequest(http.MethodGet, "/manage/film/class/find?id=999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestUpdateFilmClass_RejectsBadJSON(t *testing.T) {
	r := gin.New()
	r.POST("/manage/film/class/update", UpdateFilmClass)
	req := httptest.NewRequest(http.MethodPost, "/manage/film/class/update", bytes.NewReader([]byte("x")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestUpdateFilmClass_RejectsZeroId(t *testing.T) {
	r := gin.New()
	r.POST("/manage/film/class/update", UpdateFilmClass)
	body := []byte(`{"id":0,"name":"X","show":true}`)
	req := httptest.NewRequest(http.MethodPost, "/manage/film/class/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "Id")
}

func TestDelFilmClass_RequiresId(t *testing.T) {
	r := gin.New()
	r.GET("/manage/film/class/del", DelFilmClass)
	req := httptest.NewRequest(http.MethodGet, "/manage/film/class/del", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

// ===================== /manage/file/* =====================

func TestDelFile_RequiresId(t *testing.T) {
	r := gin.New()
	r.GET("/manage/file/del", DelFile)
	req := httptest.NewRequest(http.MethodGet, "/manage/file/del", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
	require.Contains(t, got["msg"], "标识")
}

func TestDelFile_RejectsNonNumericId(t *testing.T) {
	r := gin.New()
	r.GET("/manage/file/del", DelFile)
	req := httptest.NewRequest(http.MethodGet, "/manage/file/del?id=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

func TestPhotoWall_HappyPath(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()
	mock.ExpectQuery(`(?i)select count.+from .files.`).WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`(?i)select .+ from .files.`).WillReturnRows(
		sqlmock.NewRows([]string{"id"}))

	r := gin.New()
	r.GET("/manage/file/list", PhotoWall)
	req := httptest.NewRequest(http.MethodGet, "/manage/file/list?current=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, got["code"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPhotoWall_RejectsBadCurrent(t *testing.T) {
	r := gin.New()
	r.GET("/manage/file/list", PhotoWall)
	req := httptest.NewRequest(http.MethodGet, "/manage/file/list?current=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	got := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, got["code"])
}

// ===================== /manage/user/info (复用 UserInfo, 已在 public_routes_test.go 覆盖) =====================
