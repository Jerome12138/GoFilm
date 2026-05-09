package system

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"server/plugin/db"
)

// newGormWithMock 返回挂在 sqlmock 之上的 *gorm.DB
func newGormWithMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	sqldb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	g, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqldb,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		_ = sqldb.Close()
		t.Fatalf("gorm.Open: %v", err)
	}
	cleanup := func() {
		_ = sqldb.Close()
	}
	return g, mock, cleanup
}

// withMockDB swap 全局 db.Mdb 为 sqlmock 实例, 返回 mock 与还原函数
func withMockDB(t *testing.T) (sqlmock.Sqlmock, func()) {
	t.Helper()
	g, mock, cleanup := newGormWithMock(t)
	prev := db.Mdb
	db.Mdb = g
	return mock, func() {
		db.Mdb = prev
		cleanup()
	}
}

// withMiniRedis swap 全局 db.Rdb 为 miniredis 客户端, 返回 mr 与还原函数
func withMiniRedis(t *testing.T) (*miniredis.Miniredis, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	prev := db.Rdb
	db.Rdb = cli
	// 测试间互相隔离, 复位 SaveSearchTag 内存初始化标记
	searchTagInitMu.Lock()
	searchTagInitMap = make(map[string]struct{})
	searchTagInitMu.Unlock()
	return mr, func() {
		_ = cli.Close()
		mr.Close()
		db.Rdb = prev
	}
}
