package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"server/logic"
	"server/model/system"
)

// M3u8Proxy GET /m3u8/proxy?src=<urlencoded>
// 拉取上游 m3u8, 过滤疑似广告 segment 后以 application/vnd.apple.mpegurl 直接吐回.
// 失败一律走业务码 (200 + code=-1), 与项目其它接口风格一致, 让前端拦截器统一处理.
func M3u8Proxy(c *gin.Context) {
	src := c.Query("src")
	if src == "" {
		system.Failed("缺少 src 参数", c)
		return
	}
	filtered, err := logic.M3UL.FetchAndFilter(src)
	if err != nil {
		system.Failed("m3u8 拉取失败: "+err.Error(), c)
		return
	}
	c.Header("Cache-Control", "public, max-age=600")
	c.Data(http.StatusOK, "application/vnd.apple.mpegurl; charset=utf-8", []byte(filtered))
}
