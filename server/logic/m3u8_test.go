package logic

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"server/plugin/db"
)

// stubRedis 起一个内存 redis 并劫持 db.Rdb, 测试结束自动还原.
func stubRedis(t *testing.T) func() {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	prev := db.Rdb
	db.Rdb = cli
	return func() {
		_ = cli.Close()
		mr.Close()
		db.Rdb = prev
	}
}

// TestFilterAds_DropsOutlierSegment: 6 个 segment, 中位时长 10s + URL 长度 ~25,
// 在中间插一个 duration=2.5s、URL 极短的"广告" segment, 应当被剔除.
func TestFilterAds_DropsOutlierSegment(t *testing.T) {
	base, _ := url.Parse("https://cdn.example.com/movie/")
	body := strings.Join([]string{
		"#EXTM3U",
		"#EXT-X-VERSION:3",
		"#EXT-X-TARGETDURATION:10",
		"#EXTINF:10.0,",
		"seg-aaaaaaaaaaaaaa-001.ts",
		"#EXTINF:10.0,",
		"seg-aaaaaaaaaaaaaa-002.ts",
		"#EXTINF:10.0,",
		"seg-aaaaaaaaaaaaaa-003.ts",
		"#EXTINF:2.5,", // 广告: 时长极短
		"ad.ts",        // 广告: URL 极短
		"#EXTINF:10.0,",
		"seg-aaaaaaaaaaaaaa-004.ts",
		"#EXTINF:10.0,",
		"seg-aaaaaaaaaaaaaa-005.ts",
		"#EXTINF:10.0,",
		"seg-aaaaaaaaaaaaaa-006.ts",
		"#EXT-X-ENDLIST",
		"",
	}, "\n")

	got := filterAds(body, base)
	require.NotContains(t, got, "ad.ts", "广告 segment 必须被剔除")
	require.NotContains(t, got, "#EXTINF:2.5", "广告对应的 EXTINF 也要剔除")
	// 正常 segment 必须被绝对化
	require.Contains(t, got, "https://cdn.example.com/movie/seg-aaaaaaaaaaaaaa-001.ts")
	// 总 segment 数 = 6
	require.Equal(t, 6, strings.Count(got, "seg-aaaaaaaaaaaaaa"))
}

// TestFilterAds_DoesNotDropWhenOnlyDurationDiffers: 时长有差但 URL 长度一致,
// 单信号不足以判定广告, 不应剔除.
func TestFilterAds_DoesNotDropWhenOnlyDurationDiffers(t *testing.T) {
	base, _ := url.Parse("https://cdn.example.com/")
	body := strings.Join([]string{
		"#EXTM3U",
		"#EXTINF:10.0,", "seg-uniform-name-001.ts",
		"#EXTINF:10.0,", "seg-uniform-name-002.ts",
		"#EXTINF:10.0,", "seg-uniform-name-003.ts",
		"#EXTINF:2.0,", "seg-uniform-name-004.ts", // 仅时长离群
		"#EXTINF:10.0,", "seg-uniform-name-005.ts",
		"#EXT-X-ENDLIST", "",
	}, "\n")

	got := filterAds(body, base)
	require.Contains(t, got, "seg-uniform-name-004.ts", "单信号离群不应剔除, 保守优先")
}

// TestFilterAds_SmallSamplePassthrough: 样本少于 4 个, 直接退化为仅做 URL 绝对化.
func TestFilterAds_SmallSamplePassthrough(t *testing.T) {
	base, _ := url.Parse("https://cdn.example.com/path/")
	body := strings.Join([]string{
		"#EXTM3U",
		"#EXTINF:10.0,", "a.ts",
		"#EXTINF:10.0,", "b.ts",
		"#EXT-X-ENDLIST", "",
	}, "\n")
	got := filterAds(body, base)
	require.Contains(t, got, "https://cdn.example.com/path/a.ts")
	require.Contains(t, got, "https://cdn.example.com/path/b.ts")
}

// TestFilterAds_AbsoluteURLsUntouched: 已经是绝对 URL 的 segment, baseURL.Parse 仍返回原值, 不被破坏.
func TestFilterAds_AbsoluteURLsUntouched(t *testing.T) {
	base, _ := url.Parse("https://a.example.com/")
	body := strings.Join([]string{
		"#EXTM3U",
		"#EXTINF:10.0,", "https://b.example.com/seg-001.ts",
		"#EXTINF:10.0,", "https://b.example.com/seg-002.ts",
		"#EXTINF:10.0,", "https://b.example.com/seg-003.ts",
		"#EXTINF:10.0,", "https://b.example.com/seg-004.ts",
		"#EXT-X-ENDLIST", "",
	}, "\n")
	got := filterAds(body, base)
	require.Contains(t, got, "https://b.example.com/seg-001.ts")
}

// TestExtractFirstVariant: master playlist 提取首个 variant 的 URL.
func TestExtractFirstVariant(t *testing.T) {
	body := strings.Join([]string{
		"#EXTM3U",
		"#EXT-X-STREAM-INF:BANDWIDTH=1280000,RESOLUTION=720x480",
		"720/index.m3u8",
		"#EXT-X-STREAM-INF:BANDWIDTH=2560000,RESOLUTION=1280x720",
		"720p/index.m3u8",
	}, "\n")
	require.Equal(t, "720/index.m3u8", extractFirstVariant(body))
}

// TestFetchAndFilter_FollowsMasterPlaylist: 上游返回 master, 跟随一层抓 media playlist.
func TestFetchAndFilter_FollowsMasterPlaylist(t *testing.T) {
	defer stubRedis(t)()

	media := strings.Join([]string{
		"#EXTM3U", "#EXT-X-VERSION:3",
		"#EXTINF:10.0,", "v1.ts",
		"#EXTINF:10.0,", "v2.ts",
		"#EXTINF:10.0,", "v3.ts",
		"#EXTINF:10.0,", "v4.ts",
		"#EXT-X-ENDLIST", "",
	}, "\n")

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/master.m3u8":
			master := strings.Join([]string{
				"#EXTM3U",
				"#EXT-X-STREAM-INF:BANDWIDTH=1280000",
				"media.m3u8",
			}, "\n")
			_, _ = w.Write([]byte(master))
		case "/media.m3u8":
			_, _ = w.Write([]byte(media))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	got, err := M3UL.FetchAndFilter(srv.URL + "/master.m3u8")
	require.NoError(t, err)
	require.Contains(t, got, srv.URL+"/v1.ts", "media playlist 内 segment 必须用 media playlist 的 URL 作 base")
	require.Contains(t, got, srv.URL+"/v4.ts")
}

// TestFetchAndFilter_RejectsNonHTTP: 非 http/https URL 一律拒绝.
func TestFetchAndFilter_RejectsNonHTTP(t *testing.T) {
	defer stubRedis(t)()
	_, err := M3UL.FetchAndFilter("file:///etc/passwd")
	require.Error(t, err)
}

// TestFetchAndFilter_UpstreamErrorReturns: 上游 404 → 报错, 不缓存.
func TestFetchAndFilter_UpstreamErrorReturns(t *testing.T) {
	defer stubRedis(t)()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	_, err := M3UL.FetchAndFilter(srv.URL + "/x.m3u8")
	require.Error(t, err)
}

// TestFetchAndFilter_CacheHit: 第二次请求命中缓存, 不再回源.
func TestFetchAndFilter_CacheHit(t *testing.T) {
	defer stubRedis(t)()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body := strings.Join([]string{
			"#EXTM3U",
			"#EXTINF:10.0,", "a-segment-001.ts",
			"#EXTINF:10.0,", "a-segment-002.ts",
			"#EXTINF:10.0,", "a-segment-003.ts",
			"#EXTINF:10.0,", "a-segment-004.ts",
			"#EXT-X-ENDLIST", "",
		}, "\n")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	first, err := M3UL.FetchAndFilter(srv.URL + "/p.m3u8")
	require.NoError(t, err)
	second, err := M3UL.FetchAndFilter(srv.URL + "/p.m3u8")
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, 1, calls, "命中缓存后不应再回源")
}

// TestMedianFloat
func TestMedianFloat(t *testing.T) {
	require.Equal(t, 0.0, medianFloat(nil))
	require.Equal(t, 5.0, medianFloat([]float64{5}))
	require.Equal(t, 3.0, medianFloat([]float64{1, 3, 5}))
	require.Equal(t, 2.5, medianFloat([]float64{1, 2, 3, 4}))
}
