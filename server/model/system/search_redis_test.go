package system

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"server/config"
	"server/plugin/db"
)

func TestHandleSearchTags_IncrementsExistingScore(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()

	key := "tags:test:plot"
	HandleSearchTags("武侠,古装", key)

	got, err := db.Rdb.ZScore(db.Cxt, key, "武侠:武侠").Result()
	require.NoError(t, err)
	require.Equal(t, 1.0, got, "first invocation should set score to 1")

	HandleSearchTags("武侠,古装", key)

	got, err = db.Rdb.ZScore(db.Cxt, key, "武侠:武侠").Result()
	require.NoError(t, err)
	require.Equal(t, 2.0, got, "second invocation should incr to 2")
}

func TestHandleSearchTags_HandlesMultipleSeparators(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()

	cases := []struct {
		input string
		key   string
		want  []string
	}{
		{"武侠/古装", "k1", []string{"武侠:武侠", "古装:古装"}},
		{"动作，科幻", "k2", []string{"动作:动作", "科幻:科幻"}},
		{"恐怖、悬疑", "k3", []string{"恐怖:恐怖", "悬疑:悬疑"}},
		{"剧情", "k4", []string{"剧情:剧情"}}, // 单个
	}
	for _, c := range cases {
		HandleSearchTags(c.input, c.key)
		for _, m := range c.want {
			score, err := db.Rdb.ZScore(db.Cxt, c.key, m).Result()
			require.NoError(t, err, "key=%s member=%s", c.key, m)
			require.Equal(t, 1.0, score, "key=%s member=%s", c.key, m)
		}
	}
}

func TestHandleSearchTags_EmptyOrOther(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()

	HandleSearchTags("", "k") // 空串不应写入
	c, err := db.Rdb.ZCard(db.Cxt, "k").Result()
	require.NoError(t, err)
	require.Equal(t, int64(0), c)

	HandleSearchTags("其它", "k2") // "其它" 是占位 score=0
	score, err := db.Rdb.ZScore(db.Cxt, "k2", "其它:其它").Result()
	require.NoError(t, err)
	require.Equal(t, 0.0, score)
}

func TestSaveSearchTag_StaticFieldsInitOnceAndIncrementDynamic(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()

	s := SearchInfo{Pid: 1, ClassTag: "武侠,古装", Area: "中国大陆", Language: "国语"}

	SaveSearchTag(s)
	SaveSearchTag(s) // 第二次调用, 静态字段不应再写入, 动态字段累加

	// 静态字段: Year 至少有 12 个 member
	yearKey := fmt.Sprintf(config.SearchTag, int64(1), "Year")
	yc, _ := db.Rdb.ZCard(db.Cxt, yearKey).Result()
	require.Equal(t, int64(12), yc, "Year tags should be initialized exactly once")

	// 动态字段 Plot: 武侠:武侠 应该被累加到 2
	plotKey := fmt.Sprintf(config.SearchTag, int64(1), "Plot")
	score, err := db.Rdb.ZScore(db.Cxt, plotKey, "武侠:武侠").Result()
	require.NoError(t, err)
	require.Equal(t, 2.0, score)

	// title key 必然存在 (HMSet 一次性写入)
	titleKey := fmt.Sprintf(config.SearchTitle, int64(1))
	titleVal, _ := db.Rdb.HGet(db.Cxt, titleKey, "Plot").Result()
	require.Equal(t, "剧情", titleVal)
}

func TestScanAndDelete_RemovesByPattern(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()

	for i := 0; i < 20; i++ {
		require.NoError(t, db.Rdb.Set(db.Cxt, fmt.Sprintf("Movie:Cid1:Id%d", i), "x", 0).Err())
	}
	require.NoError(t, db.Rdb.Set(db.Cxt, "Other:keep", "y", 0).Err())

	scanAndDelete("Movie:*", 5)

	remain, _ := db.Rdb.Keys(db.Cxt, "Movie:*").Result()
	require.Equalf(t, 0, len(remain), "expected all Movie:* keys deleted, remain=%v", remain)
	exists, _ := db.Rdb.Exists(db.Cxt, "Other:keep").Result()
	require.Equal(t, int64(1), exists, "non-matching key should be preserved")
}
