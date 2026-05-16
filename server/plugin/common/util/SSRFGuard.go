package util

import (
	"errors"
	"net"
	"net/url"
)

/*
SSRF 防御工具.

适用场景:
  - 任何接受用户/外部数据控制的 URL 后, 服务端再去拉取该 URL 的代码路径
    (m3u8 代理 / spider 影片源采集 / 在线图片下载)

策略:
  - 拒绝 loopback / RFC1918 / link-local (含 169.254.169.254 元数据) /
    multicast / unspecified / RFC6598 CGNAT / IPv6 ULA
  - 对域名进行 DNS 解析后逐 IP 校验 (注意: 这只是第一道, 攻击者可 DNS rebinding;
    需要 Dialer.Control 那一层 SSRF 防御的, 见 logic/M3u8Logic.go)

为什么单独成包: spider / m3u8 / 在线图片下载都要复用. 不放在 logic 包是为了
避免 util → logic 反向依赖.
*/

// ErrBlockedHost 命中 SSRF 黑名单. 调用方应回业务级"不允许的目标地址"文案,
// 不要把具体 IP 等细节透传给客户端 (防变成内网扫描器).
var ErrBlockedHost = errors.New("blocked host")

// AllowLoopbackForTest 仅供测试用. 翻开后允许命中 127.0.0.1 监听 (httptest 起的本地 server).
// 生产代码不要改, 翻开就等于关闭整套 SSRF 防御.
var AllowLoopbackForTest = false

// ValidatePublicURL 校验 URL 解析出的所有 IP 都在公网可达范围.
// 任一 IP 命中黑名单即拒.
func ValidatePublicURL(u *url.URL) error {
	if u == nil {
		return ErrBlockedHost
	}
	host := u.Hostname()
	if host == "" {
		return ErrBlockedHost
	}
	if ip := net.ParseIP(host); ip != nil {
		if !IsPublicIP(ip) {
			return ErrBlockedHost
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return ErrBlockedHost
	}
	for _, ip := range ips {
		if !IsPublicIP(ip) {
			return ErrBlockedHost
		}
	}
	return nil
}

// IsPublicIP 判断 IP 是否属于公网可达地址.
// IsPrivate 覆盖 RFC1918 + RFC4193 私有段;
// 额外拒 loopback / link-local / multicast / unspecified / CGNAT 100.64/10.
func IsPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if !AllowLoopbackForTest && ip.IsLoopback() {
		return false
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsPrivate() {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		// RFC6598 CGNAT 100.64.0.0/10
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return false
		}
	}
	return true
}
