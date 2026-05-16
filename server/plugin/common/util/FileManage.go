package util

import (
	"bufio"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"server/config"
)

/*
数据请求保存,文件读写.

安全约束 (SSRF + 文件落盘):
  - SaveOnlineFile 仅放行 http/https + 公网可达 URL
  - 不再使用 filepath.Base(url) 作为落盘文件名 (Windows 下含 query 不剥, Linux 下被截到 ?
    之前; 一旦 URL path 形如 /image.svg?fake=.html 会落盘成可执行/可注入文件)
  - 落盘文件名固定为 random + 内容嗅探出的 ext, 不信任 URL
*/

// 与 FileController 中的图片白名单同口径
var saveOnlineMIME2Ext = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

// SaveOnlineFile 保存网络文件; 仅放行图片类型. 返回落盘后的文件名 (相对 dir).
func SaveOnlineFile(rawURL, dir string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errors.New("SaveOnlineFile: 仅支持 http/https URL")
	}
	if err := ValidatePublicURL(u); err != nil {
		return "", err
	}

	r := &RequestInfo{Uri: rawURL}
	ApiGet(r)
	if len(r.Resp) <= 0 {
		return "", errors.New("SaveOnlineFile: 响应为空")
	}

	// 内容嗅探, 拒绝非图片
	mime := http.DetectContentType(r.Resp[:min(512, len(r.Resp))])
	if idx := strings.Index(mime, ";"); idx >= 0 {
		mime = mime[:idx]
	}
	ext, ok := saveOnlineMIME2Ext[strings.TrimSpace(mime)]
	if !ok {
		return "", errors.New("SaveOnlineFile: 内容不是允许的图片类型")
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return "", err
		}
	}
	// 文件名: random8 + 嗅探 ext; 与 URL 路径完全脱钩, 避免路径穿越/扩展名伪造
	fileName := RandomString(8) + ext
	full := filepath.Join(dir, fileName)
	f, err := os.Create(full)
	if err != nil {
		return "", err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	if _, err := w.Write(r.Resp); err != nil {
		return "", err
	}
	if err := w.Flush(); err != nil {
		return "", err
	}
	return fileName, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func CreateBaseDir() error {
	if _, err := os.Stat(config.FilmPictureUploadDir); os.IsNotExist(err) {
		return os.MkdirAll(config.FilmPictureUploadDir, os.ModePerm)
	}
	return nil
}

func RemoveFile(path string) error {
	return os.Remove(path)
}
