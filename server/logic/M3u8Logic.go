package logic

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

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
*/

const (
	m3u8CacheKeyFmt = "m3u8:proxy:%s"
	m3u8CacheTTL    = 10 * time.Minute
	m3u8MaxBytes    = 2 * 1024 * 1024 // 单个 m3u8 上限
	m3u8FollowDepth = 1               // master → media playlist 仅跟一层

	// adDurationDeltaRatio: segment 时长相对中位数的偏差阈值 (0.25 = ±25%)
	adDurationDeltaRatio = 0.25
	// adURLLenDeltaRatio: segment URL 字符长度相对中位数的偏差阈值 (0.20 = ±20%)
	adURLLenDeltaRatio = 0.20
	// adMinSegments: 样本量过小时启发不可靠, 直接退化为仅做 URL 绝对化
	adMinSegments = 4
)

var (
	m3u8Client = &http.Client{Timeout: 5 * time.Second}
	reExtinf   = regexp.MustCompile(`^#EXTINF:([0-9.]+)`)
)

type M3u8Logic struct{}

var M3UL *M3u8Logic

// m3u8Segment 表示 media playlist 中的一段切片 (EXTINF + 可选 tag + URL).
type m3u8Segment struct {
	indices  []int   // 占用的原始行号 (用于剔除时整组删)
	url      string  // segment URL 原文 (绝对/相对未定)
	duration float64 // EXTINF 解析秒数; 0 表示解析失败
}

// FetchAndFilter 拉取 src 指向的 m3u8, 剔除疑似广告片段, 返回改写后的 m3u8 文本.
// src 必须是 http/https URL; 返回结果的 segment URL 已经绝对化, 播放器可直接消费.
func (m *M3u8Logic) FetchAndFilter(src string) (string, error) {
	u, err := url.Parse(src)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errors.New("src 必须是 http/https URL")
	}

	cacheKey := fmt.Sprintf(m3u8CacheKeyFmt, sha1Hex(src))
	if cached, err := db.Rdb.Get(db.Cxt, cacheKey).Result(); err == nil && cached != "" {
		return cached, nil
	} else if err != nil && !errors.Is(err, redis.Nil) {
		// 缓存层故障不致命, 走 fallthrough 重新抓; 错误吞掉避免反复打日志
	}

	body, baseURL, err := m.fetchFollow(src, m3u8FollowDepth)
	if err != nil {
		return "", err
	}
	filtered := filterAds(body, baseURL)

	// 缓存写失败也不影响返回
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
		// 进入 segment: 见到 EXTINF
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
		// 在 segment 内: # 开头的 tag 也归入当前 group (如 #EXT-X-DISCONTINUITY / #EXT-X-BYTERANGE)
		if strings.HasPrefix(trimmed, "#") {
			cur.indices = append(cur.indices, i)
			continue
		}
		// 空行: 不属于 group, 但也不结束 segment, 等下一行
		if trimmed == "" {
			continue
		}
		// 非 #, 非空: 这就是 URL 行, segment 结束
		cur.indices = append(cur.indices, i)
		cur.url = trimmed
		segs = append(segs, *cur)
		cur = nil
	}

	if len(segs) < adMinSegments {
		return rewriteURLs(lines, baseURL, nil)
	}

	dMed := medianFloat(extractDurations(segs))
	lMed := medianInt(extractURLLengths(segs))
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

func extractDurations(segs []m3u8Segment) []float64 {
	out := make([]float64, len(segs))
	for i, s := range segs {
		out[i] = s.duration
	}
	return out
}

func extractURLLengths(segs []m3u8Segment) []int {
	out := make([]int, len(segs))
	for i, s := range segs {
		out[i] = len(s.url)
	}
	return out
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
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GoFilm-m3u8-proxy/1.0")
	resp, err := m3u8Client.Do(req)
	if err != nil {
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

func sha1Hex(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
