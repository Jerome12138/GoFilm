package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"server/config"
	"server/plugin/db"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func withMiniRedis(t *testing.T) func() {
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

// TestFilmDetail_BadIdReturns400Like 调用 FilmDetail, 不带 id 返回业务错误码 (200 + code=-1)
func TestFilmDetail_BadIdReturnsBusinessFailure(t *testing.T) {
	defer withMiniRedis(t)()
	r := gin.New()
	r.GET("/filmDetail", FilmDetail)

	req := httptest.NewRequest(http.MethodGet, "/filmDetail?id=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, body["code"])
	require.Contains(t, body["msg"], "请求异常")
}

// TestSearchFilm_NoMatch 关键字未命中时, 应返回业务失败 + 提示
func TestSearchFilm_NoMatch(t *testing.T) {
	defer withMiniRedis(t)()
	// 缺 mysql, SearchFilmKeyword 内部会 panic? 看实现: 用 db.Mdb.Model(...).Count.
	// db.Mdb 为 nil 会 panic; 跳过此场景或设置 nil-tolerant 实现.
	t.Skip("requires mysql mock; covered by model layer SQL test")
}

// TestFilmClassify_MissingPidReturnsFailure
func TestFilmClassify_MissingPidReturnsFailure(t *testing.T) {
	defer withMiniRedis(t)()
	r := gin.New()
	r.GET("/filmClassify", FilmClassify)

	req := httptest.NewRequest(http.MethodGet, "/filmClassify", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, body["code"])
	require.Contains(t, body["msg"], "主分类信息")
}

// TestFilmTagSearch_MissingPidReturnsFailure
func TestFilmTagSearch_MissingPidReturnsFailure(t *testing.T) {
	defer withMiniRedis(t)()
	r := gin.New()
	r.GET("/filmClassifySearch", FilmTagSearch)

	req := httptest.NewRequest(http.MethodGet, "/filmClassifySearch", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, body["code"])
	require.Contains(t, body["msg"], "缺少分类信息")
}

// TestIndex_CacheHitReturnsCachedPayload
func TestIndex_CacheHitReturnsCachedPayload(t *testing.T) {
	defer withMiniRedis(t)()
	cached := map[string]interface{}{"category": "from-cache", "content": []interface{}{}}
	raw, _ := json.Marshal(cached)
	require.NoError(t, db.Rdb.Set(db.Cxt, config.IndexCacheKey, raw, 0).Err())

	r := gin.New()
	r.GET("/index", Index)

	req := httptest.NewRequest(http.MethodGet, "/index", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, 0, body["code"])
	data := body["data"].(map[string]interface{})
	require.Equal(t, "from-cache", data["category"])
}

// TestCategoriesInfo_EmptyTreeReturnsFailure
func TestCategoriesInfo_EmptyTreeReturnsFailure(t *testing.T) {
	defer withMiniRedis(t)()
	r := gin.New()
	r.GET("/navCategory", CategoriesInfo)

	req := httptest.NewRequest(http.MethodGet, "/navCategory", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w.Body.Bytes())
	require.EqualValues(t, -1, body["code"])
}

func decodeBody(t *testing.T, b []byte) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &body))
	return body
}
