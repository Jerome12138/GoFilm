package system

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"server/plugin/db"
)

// TestFillBasicInfoPics_BatchReplaces 一次 SELECT 拉回所有 fileinfo, 内存 map 替换 Picture
func TestFillBasicInfoPics_BatchReplaces(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`(?i)select.+from .files.\s+where relevance_id in`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "link", "relevance_id"}).
			AddRow(1, "/local/p1.png", 100).
			AddRow(2, "/local/p2.png", 200),
	)

	list := []MovieBasicInfo{
		{Id: 100, Picture: "remote-1.png"},
		{Id: 200, Picture: "remote-2.png"},
		{Id: 300, Picture: "remote-3.png"}, // 无 fileinfo, 保留原值
	}
	FillBasicInfoPics(list)
	require.Equal(t, "/local/p1.png", list[0].Picture)
	require.Equal(t, "/local/p2.png", list[1].Picture)
	require.Equal(t, "remote-3.png", list[2].Picture)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestFillBasicInfoPics_EmptyShortCircuit 空列表不应触发任何 SQL
func TestFillBasicInfoPics_EmptyShortCircuit(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()
	FillBasicInfoPics(nil)
	require.NoError(t, mock.ExpectationsWereMet(), "no SQL should be issued")
}

// TestFillBasicInfoPics_NilMdbSafe db.Mdb 未初始化时不 panic, list 保持不变
func TestFillBasicInfoPics_NilMdbSafe(t *testing.T) {
	prev := db.Mdb
	db.Mdb = nil
	defer func() { db.Mdb = prev }()

	list := []MovieBasicInfo{{Id: 1, Picture: "x.png"}}
	require.NotPanics(t, func() { FillBasicInfoPics(list) })
	require.Equal(t, "x.png", list[0].Picture)
}
