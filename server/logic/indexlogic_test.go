package logic

import (
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
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

// withMiniRedis 在 logic 包内复用一份, 避免跨包共享 helper.
func withMiniRedis(t *testing.T) (*miniredis.Miniredis, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	prev := db.Rdb
	db.Rdb = cli
	return mr, func() {
		_ = cli.Close()
		mr.Close()
		db.Rdb = prev
	}
}

// 验证 IndexPage 命中缓存时不再触发后端 (无 mysql mock 仍能返回缓存).
func TestIndexPage_CacheHit(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()

	cached := map[string]interface{}{
		"category": "fake-category",
		"content":  []interface{}{},
	}
	raw, _ := json.Marshal(cached)
	require.NoError(t, db.Rdb.Set(db.Cxt, config.IndexCacheKey, raw, 0).Err())

	il := &IndexLogic{}
	got := il.IndexPage()
	require.NotNil(t, got)
	require.Equal(t, "fake-category", got["category"])
}

// GetPidCategory 在 redis 无 CategoryTree 数据时应返回 nil 而非 panic.
func TestGetPidCategory_NoCategoryTreeReturnsNil(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()
	il := &IndexLogic{}
	require.Nil(t, il.GetPidCategory(1))
}

// GetPidCategory 命中: 写入 tree 后能找到指定 pid.
func TestGetPidCategory_Found(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()

	tree := system.CategoryTree{
		Category: &system.Category{Id: 0, Name: "root"},
		Children: []*system.CategoryTree{
			{Category: &system.Category{Id: 1, Name: "电影", Show: true}},
			{Category: &system.Category{Id: 2, Name: "电视剧", Show: true}},
		},
	}
	require.NoError(t, system.SaveCategoryTree(&tree))

	il := &IndexLogic{}
	got := il.GetPidCategory(2)
	require.NotNil(t, got)
	require.Equal(t, "电视剧", got.Name)
}

// GetFilmDetail 影片不存在时应返回零值 (不 panic, 不返回脏数据)
func TestGetFilmDetail_NotFoundReturnsEmpty(t *testing.T) {
	mock, cleanupDB := withMockDB(t)
	defer cleanupDB()
	_, cleanupRedis := withMiniRedis(t)
	defer cleanupRedis()

	mock.ExpectQuery(`(?i)select.*from .search.`).WillReturnError(gorm.ErrRecordNotFound)

	il := &IndexLogic{}
	res := il.GetFilmDetail(99999)
	require.Equal(t, system.MovieDetailVo{}, res, "missing film should return zero value, not panic")
}

// SearchTags 在无数据时返回空 map (不 panic)
func TestSearchTags_EmptyRedisReturnsBootstrap(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()
	il := &IndexLogic{}
	res := il.SearchTags(1)
	require.NotNil(t, res)
	// 始终至少包含 sortList key
	require.Contains(t, res, "sortList")
}
