package logic

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"server/plugin/common/util"
	"server/plugin/db"
)

// stubRedis 起一个内存 redis 并劫持 db.Rdb, 同时翻开 util.AllowLoopbackForTest 让 httptest 监听可达.
// 测试结束自动还原两者.
func stubRedis(t *testing.T) func() {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	prev := db.Rdb
	db.Rdb = cli
	prevLoop := util.AllowLoopbackForTest
	util.AllowLoopbackForTest = true
	return func() {
		_ = cli.Close()
		mr.Close()
		db.Rdb = prev
		util.AllowLoopbackForTest = prevLoop
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

// =========================== SSRF 防御 ===========================

// TestIsPublicIP 单元测试: 各类内网/特殊地址必须被拒.
func TestIsPublicIP(t *testing.T) {
	// 生产模式 (默认): loopback 也要拒
	prev := util.AllowLoopbackForTest
	util.AllowLoopbackForTest = false
	defer func() { util.AllowLoopbackForTest = prev }()

	cases := []struct {
		ip   string
		want bool
		desc string
	}{
		{"127.0.0.1", false, "ipv4 loopback"},
		{"127.255.255.254", false, "ipv4 loopback range"},
		{"::1", false, "ipv6 loopback"},
		{"10.0.0.1", false, "RFC1918 10/8"},
		{"172.16.0.1", false, "RFC1918 172.16/12"},
		{"172.31.255.255", false, "RFC1918 172.31"},
		{"192.168.1.1", false, "RFC1918 192.168/16"},
		{"169.254.169.254", false, "云元数据 link-local"},
		{"100.64.0.1", false, "CGNAT 100.64/10"},
		{"100.127.0.1", false, "CGNAT 100.127"},
		{"0.0.0.0", false, "unspecified"},
		{"224.0.0.1", false, "ipv4 multicast"},
		{"fc00::1", false, "ipv6 ULA"},
		{"fe80::1", false, "ipv6 link-local"},
		{"ff02::1", false, "ipv6 multicast"},
		{"8.8.8.8", true, "公网 ipv4"},
		{"1.1.1.1", true, "公网 ipv4"},
		{"100.63.255.255", true, "100.63 (CGNAT 边界外)"},
		{"100.128.0.1", true, "100.128 (CGNAT 边界外)"},
		{"2606:4700::1111", true, "公网 ipv6"},
	}
	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		require.NotNil(t, ip, "parse ip: %s", c.ip)
		got := util.IsPublicIP(ip)
		require.Equal(t, c.want, got, "%s [%s]", c.ip, c.desc)
	}
}

// TestValidatePublicURL_RejectsInternalIPLiteral: URL 里写死的内网 IP 字面量直接拒.
func TestValidatePublicURL_RejectsInternalIPLiteral(t *testing.T) {
	prev := util.AllowLoopbackForTest
	util.AllowLoopbackForTest = false
	defer func() { util.AllowLoopbackForTest = prev }()

	for _, raw := range []string{
		"http://127.0.0.1/x.m3u8",
		"http://10.0.0.1/x.m3u8",
		"http://169.254.169.254/latest/meta-data/",
		"http://[::1]/x.m3u8",
		"http://192.168.1.1:8080/x.m3u8",
	} {
		u, err := url.Parse(raw)
		require.NoError(t, err)
		require.ErrorIs(t, util.ValidatePublicURL(u), ErrBlockedHost, raw)
	}
}

// TestValidatePublicURL_AcceptsPublic: 公网 IP 字面量正常通过.
func TestValidatePublicURL_AcceptsPublic(t *testing.T) {
	prev := util.AllowLoopbackForTest
	util.AllowLoopbackForTest = false
	defer func() { util.AllowLoopbackForTest = prev }()

	u, _ := url.Parse("http://1.1.1.1/x.m3u8")
	require.NoError(t, util.ValidatePublicURL(u))
}

// TestFetchAndFilter_RejectsLoopbackInProduction: 关闭 loopback 豁免后,
// 即使 httptest 起在 127.0.0.1, FetchAndFilter 也直接拒.
func TestFetchAndFilter_RejectsLoopbackInProduction(t *testing.T) {
	defer stubRedis(t)()
	// 覆盖 stubRedis 翻开的 loopback 豁免, 模拟生产
	util.AllowLoopbackForTest = false

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("#EXTM3U\n"))
	}))
	defer srv.Close()

	_, err := M3UL.FetchAndFilter(srv.URL + "/x.m3u8")
	require.ErrorIs(t, err, ErrBlockedHost, "loopback 必须被拦; 攻击者用 127.0.0.1 探内网就靠这道")
}

// =========================== 其它 ===========================

// TestMedianFloat
func TestMedianFloat(t *testing.T) {
	require.Equal(t, 0.0, medianFloat(nil))
	require.Equal(t, 5.0, medianFloat([]float64{5}))
	require.Equal(t, 3.0, medianFloat([]float64{1, 3, 5}))
	require.Equal(t, 2.5, medianFloat([]float64{1, 2, 3, 4}))
}
