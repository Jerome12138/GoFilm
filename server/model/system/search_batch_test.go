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

// TestSaveSearchInfo_OnDuplicateUpsert 验证单条版本改用 ON DUPLICATE KEY UPDATE
// 而不再走 SELECT COUNT + INSERT/UPDATE 双 SQL.
//  1. 不再开启显式事务 (历史 tx.Begin/Commit 已删除)
//  2. 一条 INSERT...ON DUPLICATE KEY UPDATE 完成 upsert
//  3. RowsAffected=1 (新增) 时累加 redis tag, =2 (更新) 时不累加
func TestSaveSearchInfo_OnDuplicateUpsert_InsertPath(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()
	mr, cleanupRedis := withMiniRedis(t)
	defer cleanupRedis()

	// 新增路径: 不应有 SELECT COUNT, 直接 INSERT...ON DUPLICATE; RowsAffected=1 表示插入
	mock.ExpectExec(`(?i)insert into .search.+on duplicate key update`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, SaveSearchInfo(SearchInfo{Mid: 100, Pid: 1, Cid: 6}))
	require.NoError(t, mock.ExpectationsWereMet())

	// 新增时 tag 应被写入 redis (BatchHandleSearchTag 至少写入 Search:Pid?:Title)
	keys := mr.Keys()
	require.NotEmpty(t, keys, "expect redis tag keys written on insert path")
}

func TestSaveSearchInfo_OnDuplicateUpsert_UpdatePath(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()
	mr, cleanupRedis := withMiniRedis(t)
	defer cleanupRedis()

	// 更新路径: RowsAffected=2 (MySQL ON DUPLICATE 命中更新且字段变化)
	mock.ExpectExec(`(?i)insert into .search.+on duplicate key update`).
		WillReturnResult(sqlmock.NewResult(0, 2))

	require.NoError(t, SaveSearchInfo(SearchInfo{Mid: 100, Pid: 1, Cid: 6}))
	require.NoError(t, mock.ExpectationsWereMet())

	// 更新时不应累加 tag, redis 应保持空
	require.Empty(t, mr.Keys(), "update path must not accumulate redis tags")
}

// TestHandleTagStr_MalformedMemberNoPanic 验证修复:
// 历史 bug 在 len(sl) > 0 时直接访问 sl[1], 当 ZSet member 不含 ":" 时
// strings.Split 返回 len=1, sl[1] 越界 panic 把前台 API 打 500.
// 修复后跳过格式不合法的 member, 仍返回合法部分.
func TestHandleTagStr_MalformedMemberNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("HandleTagStr panicked on malformed member: %v", r)
		}
	}()

	// "动作" 缺少 ":分隔符" → 不合法, 必须跳过
	// "喜剧:喜剧"  → 合法, 应保留
	out := HandleTagStr("Plot", "动作", "喜剧:喜剧", "", "战争:战争")

	// 头部 "全部" + 末尾 "其它" + 中间 2 条合法 = 4
	require.Len(t, out, 4)
	require.Equal(t, "全部", out[0]["Name"])
	require.Equal(t, "喜剧", out[1]["Name"])
	require.Equal(t, "喜剧", out[1]["Value"])
	require.Equal(t, "战争", out[2]["Name"])
	require.Equal(t, "其它", out[3]["Name"])
}
