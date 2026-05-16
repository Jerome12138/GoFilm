package logic

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"server/plugin/db"
)

/*
M3u8Logic 处理 m3u8 文件的拉取 + 广告片段过滤 + URL 绝对化.

设计要点:
 1. 不落盘, 全部内存处理. 上游 m3u8 抓回后改写, 直接以文本返回.
 2. 跟随一层 master playlist (#EXT-X-STREAM-INF) → media playlist, 防递归.
 3. 广告片段识别采用 "duration 离群 + URL 字符长度离群" 双信号:
    任一信号未触发就不剔除, 保守优先, 误伤代价大于漏报.
 4. media playlist 结果按 sha1(srcURL) 缓存到 Redis 10 分钟; master playlist 不缓存
    (它的字节都很小, 解析也极快, 缓存收益不大且会让 variant 选择固化).
 5. 上游请求 5s 超时, 响应体上限 2 MB; 超过任一上限直接报错, 不挂主线程.
 6. SSRF 防御 (三道):
    a) 入口域名 DNS 解析后逐 IP 校验, 拒 私网/loopback/link-local/multicast.
    b) 重定向 hook 重做同样校验, 防 "公网域名 302 到内网".
    c) Dialer.Control 在 connect 前再校验目标 IP, 抵御 DNS rebinding
       (DNS 第一次返回公网 IP 通过校验, 第二次解析返回内网 IP — 这里仍拦下).
*/

const (
	m3u8CacheKeyFmt = "m3u8:proxy:%s"
	m3u8CacheTTL    = 10 * time.Minute
	m3u8MaxBytes    = 2 * 1024 * 1024 // 单个 m3u8 上限
	m3u8FollowDepth = 1               // master → media playlist 仅跟一层
	m3u8MaxRedirect = 3               // http.Client 最多跟 3 次重定向

	adDurationDeltaRatio = 0.25
	adURLLenDeltaRatio   = 0.20
	adMinSegments        = 4
)

// ErrBlockedHost 表示目标地址解析到内网/特殊地址, 被 SSRF 防御拦下.
// 暴露给上层是为了让 controller 区分用户错误 (传错 URL) 与系统错误,
// 但具体细节不能回给客户端, 否则会变成内网扫描器.
var ErrBlockedHost = errors.New("blocked host")

var (
	m3u8Client *http.Client
	reExtinf   = regexp.MustCompile(`^#EXTINF:([0-9.]+)`)

	// m3u8AllowLoopback: 仅在测试中由 setup 翻开, 允许命中 httptest 起的 127.0.0.1 监听.
	// 生产代码不要改, 翻开就等于关闭整套 SSRF 防御.
	m3u8AllowLoopback = false
)

func init() {
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(network, address string, _ syscall.RawConn) error {
			// 在 connect 前最后一道关: 目标 IP 已由 net 包解析好,
			// 攻击者就算 DNS rebinding 让两次解析结果不同, 这里也能拦下.
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil {
				return ErrBlockedHost
			}
			if !isPublicIP(ip) {
				return ErrBlockedHost
			}
			return nil
		},
	}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          64,
		MaxIdleConnsPerHost:   8,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	}
	m3u8Client = &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= m3u8MaxRedirect {
				return errors.New("too many redirects")
			}
			// 重定向目标也必须公网可达, 防 "公网 → 302 → 内网" 绕过
			if err := validatePublicURL(req.URL); err != nil {
				return err
			}
			return nil
		},
	}
}

type M3u8Logic struct{}

var M3UL *M3u8Logic

// m3u8Segment 表示 media playlist 中的一段切片 (EXTINF + 可选 tag + URL).
type m3u8Segment struct {
	indices  []int   // 占用的原始行号 (用于剔除时整组删)
	url      string  // segment URL 原文 (绝对/相对未定)
	duration float64 // EXTINF 解析秒数; 0 表示解析失败
}

// FetchAndFilter 拉取 src 指向的 m3u8, 剔除疑似广告片段, 返回改写后的 m3u8 文本.
// src 必须是 http/https URL 且解析到公网 IP; 返回结果的 segment URL 已绝对化.
func (m *M3u8Logic) FetchAndFilter(src string) (string, error) {
	u, err := url.Parse(src)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errors.New("src 必须是 http/https URL")
	}
	if err := validatePublicURL(u); err != nil {
		return "", err
	}

	cacheKey := fmt.Sprintf(m3u8CacheKeyFmt, sha1Hex(src))
	if cached, _ := db.Rdb.Get(db.Cxt, cacheKey).Result(); cached != "" {
		return cached, nil
	}

	body, baseURL, err := m.fetchFollow(src, m3u8FollowDepth)
	if err != nil {
		return "", err
	}
	filtered := filterAds(body, baseURL)

	// 缓存写失败不致命, 忽略
	_ = db.Rdb.Set(db.Cxt, cacheKey, filtered, m3u8CacheTTL).Err()
	return filtered, nil
}

// fetchFollow 抓 m3u8 文本; 若是 master playlist (#EXT-X-STREAM-INF) 就跟随到对应 media playlist,
// 最多跟随 depth 层防递归. 返回 (文本, baseURL, err).
func (m *M3u8Logic) fetchFollow(src string, depth int) (string, *url.URL, error) {
	body, err := fetchText(src)
	if err != nil {
		return "", nil, err
	}
	baseURL, _ := url.Parse(src)
	if depth > 0 && strings.Contains(body, "#EXT-X-STREAM-INF") {
		if next := extractFirstVariant(body); next != "" {
			if resolved, err := baseURL.Parse(next); err == nil {
				if err := validatePublicURL(resolved); err != nil {
					return "", nil, err
				}
				return m.fetchFollow(resolved.String(), depth-1)
			}
		}
	}
	return body, baseURL, nil
}

// extractFirstVariant 找 master playlist 中第一个 variant 的 URL.
// EXT-X-STREAM-INF 之后第一个非 # 非空行即是.
func extractFirstVariant(body string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "#EXT-X-STREAM-INF") {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			v := strings.TrimSpace(lines[j])
			if v == "" || strings.HasPrefix(v, "#") {
				continue
			}
			return v
		}
	}
	return ""
}

// filterAds 解析 media playlist, 剔除疑似广告 segment, 并把相对 URL 改写成绝对.
//
// 算法:
//  1. 收集所有 (extinfLine, midTags..., urlLine) 三元组, 称为 segment
//  2. 计算所有 segment 的 duration 中位数 dMed, URL 长度中位数 lMed
//  3. 对每个 segment, 若同时满足 |dur-dMed|/dMed > 0.25 AND |urlLen-lMed|/lMed > 0.20
//     则视为广告, 整组行 (extinf + 中间 tag + url) 全部丢弃.
//
// 样本不足 adMinSegments 时直接退化为 rewriteOnly (只做 URL 绝对化, 不冒险剔除).
func filterAds(body string, baseURL *url.URL) string {
	lines := strings.Split(body, "\n")

	var segs []m3u8Segment
	var cur *m3u8Segment
	for i, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#EXTINF") {
			d := 0.0
			if mm := reExtinf.FindStringSubmatch(trimmed); len(mm) > 1 {
				d, _ = strconv.ParseFloat(mm[1], 64)
			}
			cur = &m3u8Segment{indices: []int{i}, duration: d}
			continue
		}
		if cur == nil {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			cur.indices = append(cur.indices, i)
			continue
		}
		if trimmed == "" {
			continue
		}
		cur.indices = append(cur.indices, i)
		cur.url = trimmed
		segs = append(segs, *cur)
		cur = nil
	}

	if len(segs) < adMinSegments {
		return rewriteURLs(lines, baseURL, nil)
	}

	durs := make([]float64, len(segs))
	lens := make([]int, len(segs))
	for i, s := range segs {
		durs[i] = s.duration
		lens[i] = len(s.url)
	}
	dMed := medianFloat(durs)
	lMed := medianInt(lens)
	if dMed <= 0 {
		return rewriteURLs(lines, baseURL, nil)
	}

	skip := make(map[int]bool)
	for _, s := range segs {
		durDelta := absFloat(s.duration-dMed) / dMed
		lenDelta := 0.0
		if lMed > 0 {
			lenDelta = absFloat(float64(len(s.url))-float64(lMed)) / float64(lMed)
		}
		if durDelta > adDurationDeltaRatio && lenDelta > adURLLenDeltaRatio {
			for _, idx := range s.indices {
				skip[idx] = true
			}
		}
	}

	return rewriteURLs(lines, baseURL, skip)
}

// rewriteURLs 输出处理后的 m3u8: 跳过 skip 中的行号, 把剩余的非 # 行绝对化.
// skip 可为 nil (表示无剔除).
func rewriteURLs(lines []string, baseURL *url.URL, skip map[int]bool) string {
	out := make([]string, 0, len(lines))
	for i, raw := range lines {
		if skip[i] {
			continue
		}
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			if abs, err := baseURL.Parse(trimmed); err == nil {
				line = abs.String()
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func medianFloat(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]float64(nil), xs...)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return (cp[n/2-1] + cp[n/2]) / 2
}

func medianInt(xs []int) int {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]int(nil), xs...)
	sort.Ints(cp)
	return cp[len(cp)/2]
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func fetchText(target string) (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GoFilm-m3u8-proxy/1.0")
	resp, err := m3u8Client.Do(req)
	if err != nil {
		// 如果是 dialer/redirect 抛的 ErrBlockedHost, 沿着 url.Error 链向上透传,
		// 让 controller 用 errors.Is 识别后回 401-like 业务码.
		if errors.Is(err, ErrBlockedHost) {
			return "", ErrBlockedHost
		}
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upstream status %d", resp.StatusCode)
	}
	lr := io.LimitReader(resp.Body, int64(m3u8MaxBytes+1))
	body, err := io.ReadAll(lr)
	if err != nil {
		return "", err
	}
	if len(body) > m3u8MaxBytes {
		return "", errors.New("m3u8 too large")
	}
	return string(body), nil
}

// validatePublicURL 校验 URL 解析到的所有 IP 都属于公网可达地址.
// 任一 IP 命中私网/loopback/link-local/multicast/unspecified 即拒.
// 域名: 走 DNS 解析全部 A/AAAA 一起判.
// 数字 IP: 直接判.
func validatePublicURL(u *url.URL) error {
	if u == nil {
		return ErrBlockedHost
	}
	host := u.Hostname()
	if host == "" {
		return ErrBlockedHost
	}
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return ErrBlockedHost
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return ErrBlockedHost
	}
	for _, ip := range ips {
		if !isPublicIP(ip) {
			return ErrBlockedHost
		}
	}
	return nil
}

// isPublicIP 判定 IP 是否属于公网可达地址.
// IsPrivate 覆盖 RFC1918 + RFC4193 的私有地址段.
func isPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if !m3u8AllowLoopback && ip.IsLoopback() {
		return false
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsPrivate() {
		return false
	}
	// 显式拒一些 IsPrivate 不覆盖的特殊段:
	// 169.254.169.254 (云元数据) — 已被 IsLinkLocalUnicast 覆盖
	// 100.64.0.0/10  (RFC6598 CGNAT, 视情况而定; 这里拒掉)
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return false
		}
	}
	return true
}

func sha1Hex(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
