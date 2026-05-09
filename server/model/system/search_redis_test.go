package system

import (
	"encoding/json"
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

// TestBatchHandleSearchTag_AggregatesSameMember 验证同 (tagKey, member) 跨多个 SearchInfo 累加合并:
// 4 部影片同样的 ClassTag="武侠,古装" → 武侠/古装 各应累加到 4
func TestBatchHandleSearchTag_AggregatesSameMember(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()

	infos := []SearchInfo{
		{Pid: 1, ClassTag: "武侠,古装", Area: "中国大陆", Language: "国语"},
		{Pid: 1, ClassTag: "武侠,古装", Area: "中国大陆", Language: "国语"},
		{Pid: 1, ClassTag: "武侠,古装", Area: "中国大陆", Language: "国语"},
		{Pid: 1, ClassTag: "武侠,古装", Area: "中国大陆", Language: "国语"},
	}
	BatchHandleSearchTag(infos...)

	plotKey := fmt.Sprintf(config.SearchTag, int64(1), "Plot")
	for _, m := range []string{"武侠:武侠", "古装:古装"} {
		score, err := db.Rdb.ZScore(db.Cxt, plotKey, m).Result()
		require.NoError(t, err)
		require.Equal(t, 4.0, score, "member=%s", m)
	}

	areaKey := fmt.Sprintf(config.SearchTag, int64(1), "Area")
	score, err := db.Rdb.ZScore(db.Cxt, areaKey, "中国大陆:中国大陆").Result()
	require.NoError(t, err)
	require.Equal(t, 4.0, score)
}

// TestBatchHandleSearchTag_StaticInitOnceAcrossPids 验证多 pid 各自的静态 tag 仅 init 一次,
// 不会因输入里出现 N 部同 pid 影片就重复 HMSet/Year ZAdd.
func TestBatchHandleSearchTag_StaticInitOnceAcrossPids(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()

	infos := []SearchInfo{
		{Pid: 1, ClassTag: "动作"},
		{Pid: 1, ClassTag: "喜剧"},
		{Pid: 2, ClassTag: "悬疑"},
	}
	BatchHandleSearchTag(infos...)

	for _, pid := range []int64{1, 2} {
		yearKey := fmt.Sprintf(config.SearchTag, pid, "Year")
		yc, _ := db.Rdb.ZCard(db.Cxt, yearKey).Result()
		require.Equal(t, int64(12), yc, "pid=%d Year init", pid)

		titleKey := fmt.Sprintf(config.SearchTitle, pid)
		titleVal, _ := db.Rdb.HGet(db.Cxt, titleKey, "Plot").Result()
		require.Equal(t, "剧情", titleVal)
	}
}

// TestBatchHandleSearchTag_OtherPlaceholderKeepsZero 验证 "其它" 占位仍以 score=0 写入 ZSet
func TestBatchHandleSearchTag_OtherPlaceholderKeepsZero(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()

	BatchHandleSearchTag(SearchInfo{Pid: 9, ClassTag: "其它"})
	plotKey := fmt.Sprintf(config.SearchTag, int64(9), "Plot")
	score, err := db.Rdb.ZScore(db.Cxt, plotKey, "其它:其它").Result()
	require.NoError(t, err)
	require.Equal(t, 0.0, score)
}

// TestBatchGetMultiplePlay_PipelinedHmgetReturnsFirstHit
// 验证 BatchGetMultiplePlay: 多个站点用 pipeline + HMGET, 每个站点取第一个非空 hit, 顺序对齐.
func TestBatchGetMultiplePlay_PipelinedHmgetReturnsFirstHit(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()

	siteA := "siteA"
	siteB := "siteB"
	siteC := "siteC" // 完全无数据
	keyMatch := "k_hit"
	keyMiss := "k_miss"

	// site A 命中 keyMatch
	plA := []MovieUrlInfo{{Episode: "01", Link: "http://a/01"}}
	rawA, _ := json.Marshal(plA)
	require.NoError(t, db.Rdb.HSet(db.Cxt, fmt.Sprintf(config.MultipleSiteDetail, siteA), keyMatch, rawA).Err())

	// site B 命中 keyMiss (顺序无关, 内部应跨字段取第一个非空)
	plB := []MovieUrlInfo{{Episode: "01", Link: "http://b/01"}, {Episode: "02", Link: "http://b/02"}}
	rawB, _ := json.Marshal(plB)
	require.NoError(t, db.Rdb.HSet(db.Cxt, fmt.Sprintf(config.MultipleSiteDetail, siteB), keyMiss, rawB).Err())

	sources := []FilmSource{{Id: siteA}, {Id: siteB}, {Id: siteC}}
	got := BatchGetMultiplePlay(sources, []string{keyMatch, keyMiss})

	require.Len(t, got, 3)
	require.Len(t, got[0], 1)
	require.Equal(t, "http://a/01", got[0][0].Link)
	require.Len(t, got[1], 2)
	require.Equal(t, "http://b/02", got[1][1].Link)
	require.Nil(t, got[2], "siteC should map to nil playList")
}

// TestBatchGetMultiplePlay_EmptyInputs 空 source 或空 keys 直接返回长度对齐的全 nil
func TestBatchGetMultiplePlay_EmptyInputs(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()

	require.Empty(t, BatchGetMultiplePlay(nil, []string{"k"}))
	require.Empty(t, BatchGetMultiplePlay([]FilmSource{}, []string{"k"}))
	got := BatchGetMultiplePlay([]FilmSource{{Id: "x"}}, nil)
	require.Len(t, got, 1)
	require.Nil(t, got[0])
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
