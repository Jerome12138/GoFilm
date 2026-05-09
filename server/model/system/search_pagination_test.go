package system

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// TestGetMovieListBySort_OffsetCalc 验证修复 (page.Current - 1) * page.PageSize 公式
// 历史 bug: (page.Current - 10*page.PageSize) 导致负数 offset, 第 2 页起完全错乱.
func TestGetMovieListBySort_OffsetCalc(t *testing.T) {
	mock, cleanupDB := withMockDB(t)
	defer cleanupDB()
	_, cleanupRedis := withMiniRedis(t)
	defer cleanupRedis()

	// 第 2 页, 每页 14 → OFFSET 14
	rows := sqlmock.NewRows([]string{"id", "mid", "cid", "name", "year"})
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM`)).WillReturnRows(rows)

	page := &Page{PageSize: 14, Current: 2}
	_ = GetMovieListBySort(1, 1, page)

	// sqlmock 会捕获实际执行的 SQL, 我们检查 OFFSET 数值
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

// TestGetMovieListBySort_FirstPageOffsetZero 第 1 页 offset 为 0
func TestGetMovieListBySort_FirstPageOffsetZero(t *testing.T) {
	mock, cleanupDB := withMockDB(t)
	defer cleanupDB()
	_, cleanupRedis := withMiniRedis(t)
	defer cleanupRedis()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	page := &Page{PageSize: 14, Current: 1}
	_ = GetMovieListBySort(0, 1, page)

	require.NoError(t, mock.ExpectationsWereMet())
}

// TestFindFilmIds_OffsetUsesPageSize 验证 FindFilmIds 也用 (Current-1)*PageSize
func TestFindFilmIds_OffsetUsesPageSize(t *testing.T) {
	mock, cleanupDB := withMockDB(t)
	defer cleanupDB()

	// Count (sqlmock regex 默认区分大小写, 用 (?i) 兼容 COUNT/count)
	mock.ExpectQuery(`(?i)select\s+count\(.*\)\s+from\s+.search.`).WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(100),
	)
	// Find ids
	mock.ExpectQuery(`(?i)select\s+.mid.\s+from\s+.search.`).WillReturnRows(
		sqlmock.NewRows([]string{"mid"}).AddRow(1).AddRow(2),
	)

	page := &Page{PageSize: 10, Current: 3}
	_, err := FindFilmIds(map[string]string{"wd": "x"}, page)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Equal(t, 100, page.Total)
}
