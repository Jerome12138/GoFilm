package controller

import (
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"server/config"
	"server/logic"
	"server/model/system"
	"server/plugin/common/util"
)

/*
文件上传安全约束:
  - 仅允许图片类型 (jpg/jpeg/png/webp/gif)
  - 扩展名 + 实际内容 (http.DetectContentType) 双判, 防伪造扩展名
  - 服务端重新生成文件名 (随机 8 位 + 白名单 ext), 不信任客户端文件名
  - 上限 10 MB / 文件
*/

const (
	maxUploadBytes = 10 * 1024 * 1024
)

var (
	// 后端最终使用的扩展名 (注意: 含点); jpeg 文件统一保存为 .jpg
	allowedImageExts = map[string]string{
		".jpg":  ".jpg",
		".jpeg": ".jpg",
		".png":  ".png",
		".webp": ".webp",
		".gif":  ".gif",
	}
	// http.DetectContentType 返回的 MIME 白名单
	allowedImageMIMEs = map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
		"image/gif":  true,
	}
)

// validateAndNormalize 校验上传文件并返回服务端最终落盘扩展名 (含点).
// 失败时返回 error, 调用方按业务码失败回吐用户友好文案 (不透传 err.Error()).
func validateAndNormalize(file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", errors.New("missing file")
	}
	if file.Size <= 0 {
		return "", errors.New("空文件")
	}
	if file.Size > maxUploadBytes {
		return "", errors.New("文件超出 10MB 上限")
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	finalExt, ok := allowedImageExts[ext]
	if !ok {
		return "", errors.New("不支持的文件类型, 仅允许 jpg/png/webp/gif")
	}
	// 嗅探实际内容前 512 字节, 防伪造扩展名
	f, err := file.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()
	head := make([]byte, 512)
	n, _ := f.Read(head)
	mime := http.DetectContentType(head[:n])
	// 去掉 charset 之类后缀
	if idx := strings.Index(mime, ";"); idx >= 0 {
		mime = mime[:idx]
	}
	if !allowedImageMIMEs[strings.TrimSpace(mime)] {
		return "", errors.New("文件内容与类型不符")
	}
	return finalExt, nil
}

// SingleUpload 单文件上传 (仅图片).
func SingleUpload(c *gin.Context) {
	v, ok := c.Get(config.AuthUserClaims)
	if !ok {
		system.Failed("上传失败, 当前用户信息异常", c)
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("upload: FormFile err: %v", err)
		system.Failed("文件参数异常", c)
		return
	}
	ext, err := validateAndNormalize(file)
	if err != nil {
		system.Failed(err.Error(), c) // 这里的 err 都是受控的中文文案, 不泄漏内部
		return
	}
	fileName := fmt.Sprintf("%s/%s%s", config.FilmPictureUploadDir, util.RandomString(8), ext)
	if err := c.SaveUploadedFile(file, fileName); err != nil {
		log.Printf("upload: SaveUploadedFile err: %v", err)
		system.Failed("文件保存失败", c)
		return
	}
	c.Header("X-Content-Type-Options", "nosniff")
	uc := v.(*system.UserClaims)
	link := logic.FileL.SingleFileUpload(fileName, int(uc.UserID))
	system.Success(link, "上传成功", c)
}

// MultipleUpload 批量文件上传 (仅图片).
func MultipleUpload(c *gin.Context) {
	v, ok := c.Get(config.AuthUserClaims)
	if !ok {
		system.Failed("上传失败, 当前用户信息异常", c)
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		log.Printf("upload: MultipartForm err: %v", err)
		system.Failed("表单解析失败", c)
		return
	}
	files := form.File["files"]
	uc := v.(*system.UserClaims)

	var fileNames []string
	for _, file := range files {
		ext, err := validateAndNormalize(file)
		if err != nil {
			system.Failed(file.Filename+": "+err.Error(), c)
			return
		}
		fileName := fmt.Sprintf("%s/%s%s", config.FilmPictureUploadDir, util.RandomString(8), ext)
		if err := c.SaveUploadedFile(file, fileName); err != nil {
			log.Printf("upload: SaveUploadedFile err: %v", err)
			system.Failed("文件保存失败", c)
			return
		}
		fileNames = append(fileNames, logic.FileL.SingleFileUpload(fileName, int(uc.UserID)))
	}
	c.Header("X-Content-Type-Options", "nosniff")
	system.Success(fileNames, "上传成功", c)
}

// DelFile 删除文件
func DelFile(c *gin.Context) {
	id, err := strconv.ParseUint(c.DefaultQuery("id", ""), 10, 64)
	if err != nil {
		system.Failed("操作失败, 未获取到需删除的文件标识信息", c)
		return
	}
	if e := logic.FileL.RemoveFileById(uint(id)); e != nil {
		log.Printf("file del err id=%d: %v", id, e)
		system.Failed("删除失败", c)
		return
	}
	system.SuccessOnlyMsg("文件已删除", c)
}

// PhotoWall 照片墙数据
func PhotoWall(c *gin.Context) {
	current, err := strconv.Atoi(c.DefaultQuery("current", "1"))
	if err != nil {
		system.Failed("图片分页数据获取失败, 分页参数异常", c)
		return
	}
	page := system.Page{PageSize: 39, Current: current}
	pl := logic.FileL.GetPhotoPage(&page)
	system.Success(gin.H{"list": pl, "page": page}, "图片分页数据获取成功", c)
}
