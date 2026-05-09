package system

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// TestBatchSave_UsesFixedBatchSize 验证 BatchSave 走单次事务 + CreateInBatches.
// 这里只断言事务结构 (BEGIN .. INSERT .. COMMIT) 与 INSERT 出现一次, 不强校验列.
func TestBatchSave_UsesFixedBatchSize(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()
	_, cleanupRedis := withMiniRedis(t)
	defer cleanupRedis()

	mock.ExpectBegin()
	mock.ExpectExec(`(?i)insert into .search.`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	BatchSave([]SearchInfo{{Mid: 1, Pid: 1, Cid: 6}})
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestBatchSave_EmptyShortCircuit 空 list 不应启动事务
func TestBatchSave_EmptyShortCircuit(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()
	BatchSave(nil)
	require.NoError(t, mock.ExpectationsWereMet(), "no SQL should be issued")
}

// TestBatchSaveOrUpdate_UpsertSinglePass 验证关键路径:
//  1. SELECT mid IN(...) 一次性拿回已存在 mid (区分新增/更新)
//  2. INSERT ... ON DUPLICATE KEY UPDATE 一次 batch upsert
//  3. 仅对新增 mid 走后续 redis tag 写入 (本测试只验事务 SQL)
func TestBatchSaveOrUpdate_UpsertSinglePass(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()
	_, cleanupRedis := withMiniRedis(t)
	defer cleanupRedis()

	mock.ExpectBegin()
	// Pluck mid IN (?, ?) — 返回已存在 mid=1
	mock.ExpectQuery(`(?i)select .mid. from .search. where mid in`).
		WithArgs(int64(1), int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"mid"}).AddRow(1))
	// INSERT ... ON DUPLICATE KEY UPDATE — GORM 会拼 ON DUPLICATE KEY UPDATE
	mock.ExpectExec(`(?i)insert into .search.+on duplicate key update`).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	BatchSaveOrUpdate([]SearchInfo{
		{Mid: 1, Pid: 1, Cid: 6},
		{Mid: 2, Pid: 1, Cid: 6},
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestBatchSaveOrUpdate_EmptyShortCircuit 空 list 不应启动事务
func TestBatchSaveOrUpdate_EmptyShortCircuit(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()
	BatchSaveOrUpdate(nil)
	require.NoError(t, mock.ExpectationsWereMet(), "no SQL should be issued")
}
