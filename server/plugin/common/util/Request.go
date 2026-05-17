package util

import (
	"fmt"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

/*
网络请求, 数据爬取

注意:
  - 历史实现使用全局 colly.Collector + 在每次 ApiGet 中调用 OnResponse 注册回调,
    导致回调链表无限累加, 长时间运行后单次响应触发 N 次回调, r.Resp 会被串到其它请求.
  - 现改为每次请求新建一个轻量 collector, 回调隔离, 不污染下次请求.
  - 所有 collector 共享同一个 *http.Transport (sharedTransport), 复用 TCP / TLS 连接池,
    避免每次 ApiGet 都新建连接 (几十万次采集 = 几十万次 TLS 握手).
*/

// RequestInfo 请求参数结构体
type RequestInfo struct {
	Uri    string      `json:"uri"`    // 请求url地址
	Params url.Values  `json:"param"`  // 请求参数
	Header http.Header `json:"header"` // 请求头数据
	Resp   []byte      `json:"resp"`   // 响应结果数据
}

const defaultRequestTimeout = 20 * time.Second

var (
	// RefererUrl 记录上次请求的 url, 多采集任务并发时由 mutex 保护
	RefererUrl   string
	refererMu    sync.RWMutex
	defaultUA    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"

	// sharedTransport 提供给所有 collector 共享, 复用 Keep-Alive 连接池.
	// MaxIdleConnsPerHost 设为 64 是因为采集场景常对同一资源站发出几十/上百次并发请求,
	// 默认值 2 会造成大量短连接, TLS 握手成为瓶颈.
	sharedTransport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   64,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
)

func getReferer() string {
	refererMu.RLock()
	defer refererMu.RUnlock()
	return RefererUrl
}

func setReferer(u string) {
	refererMu.Lock()
	RefererUrl = u
	refererMu.Unlock()
}

// CreateClient 保留用于兼容历史调用方（外部目前无引用）, 每次返回一个独立 collector.
func CreateClient() *colly.Collector {
	return newCollector(defaultRequestTimeout)
}

// newCollector 构造一个一次性 collector, 不挂任何 OnResponse, 由调用方按需注册.
// transport 共享 sharedTransport, 复用 TCP 连接池 (Keep-Alive).
func newCollector(timeout time.Duration) *colly.Collector {
	c := colly.NewCollector()
	c.MaxDepth = 1
	c.AllowURLRevisit = true
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}
	c.SetRequestTimeout(timeout)
	// 复用共享 Transport, colly 不暴露 SetTransport, 但 WithTransport 等价
	c.WithTransport(sharedTransport)
	c.OnRequest(func(req *colly.Request) {
		req.Headers.Set("Content-Type", "application/json;charset=UTF-8")
		if req.Headers.Get("User-Agent") == "" {
			req.Headers.Set("User-Agent", defaultUA)
		}
		// Referer: 仅在 host 一致时携带, 避免跨站泄漏
		if r := getReferer(); r != "" && strings.Contains(r, req.URL.Host) {
			req.Headers.Set("Referer", r)
		}
	})
	c.OnError(func(response *colly.Response, err error) {
		if response != nil && response.Request != nil {
			log.Printf("请求异常: URL: %s Error: %s\n", response.Request.URL, err)
		} else {
			log.Printf("请求异常: %s\n", err)
		}
	})
	return c
}

// buildVisitURL 把 r.Uri + r.Params 拼接成最终请求 URL, 同时做 SSRF 防御:
// 仅允许 http/https, host 必须能解析到公网 IP. 否则返回 ErrBlockedHost.
//
// spider 数据源都是 admin 在采集源里配置 + 第三方资源站返回的 URL, 都需要这道校验,
// 否则恶意采集源 URL 指向内网会变成 SSRF.
func buildVisitURL(r *RequestInfo) (string, error) {
	full := fmt.Sprintf("%s?%s", r.Uri, r.Params.Encode())
	u, err := url.Parse(full)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", ErrBlockedHost
	}
	if err := ValidatePublicURL(u); err != nil {
		return "", err
	}
	return full, nil
}

// ApiGet 请求数据的方法, 结果写入 r.Resp
func ApiGet(r *RequestInfo) {
	timeout := defaultRequestTimeout
	if r.Header != nil {
		// 注: 旧实现 if t, err := ...; err != nil && t > 0 → 条件逻辑相反, 这里改为 err == nil
		if t, err := strconv.Atoi(r.Header.Get("timeout")); err == nil && t > 0 {
			timeout = time.Duration(t) * time.Second
		}
	}
	visit, err := buildVisitURL(r)
	if err != nil {
		log.Printf("ApiGet 拒绝 URL: uri=%q err=%v", r.Uri, err)
		r.Resp = []byte{}
		return
	}
	c := newCollector(timeout)
	extensions.RandomUserAgent(c)
	c.OnResponse(func(response *colly.Response) {
		if (response.StatusCode == 200 || (response.StatusCode >= 300 && response.StatusCode <= 399)) && len(response.Body) > 0 {
			r.Resp = response.Body
		} else {
			r.Resp = []byte{}
		}
		setReferer(response.Request.URL.String())
	})
	if err := c.Visit(visit); err != nil {
		log.Println("获取数据失败: ", err)
	}
}

// ApiTest 测试 API 是否可用, 错误返回给调用方
func ApiTest(r *RequestInfo) error {
	visit, err := buildVisitURL(r)
	if err != nil {
		return err
	}
	c := newCollector(defaultRequestTimeout)
	c.OnResponse(func(response *colly.Response) {
		if (response.StatusCode == 200 || (response.StatusCode >= 300 && response.StatusCode <= 399)) && len(response.Body) > 0 {
			r.Resp = response.Body
		} else {
			r.Resp = []byte{}
		}
	})
	if err := c.Visit(visit); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

