package system

import (
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"log"
	"path/filepath"
	"regexp"
	"server/config"
	"server/plugin/common/util"
	"server/plugin/db"
	"strings"
)

// reFilePicExt 提取并去掉文件扩展名 (如 ".png"), 提前编译避免循环内重复 MustCompile.
var reFilePicExt = regexp.MustCompile(`\.[^.]+$`)

// FileInfo 图片信息对象
type FileInfo struct {
	gorm.Model
	Link        string `json:"link"`        // 图片链接
	Uid         int    `json:"uid"`         // 上传人ID
	RelevanceId int64  `json:"relevanceId"` // 关联资源ID
	Type        int    `json:"type"`        // 文件类型 (0 影片封面, 1 用户头像)
	Fid         string `json:"fid"`         // 图片唯一标识, 通常为文件名
	FileType    string `json:"fileType"`    // 文件类型, txt, png, jpg
	//Size        int    `json:"size"`        // 文件大小
}

// VirtualPicture 采集入站,待同步的图片信息
type VirtualPicture struct {
	Id   int64  `json:"id"`
	Link string `json:"link"`
}

//------------------------------------------------本地图库------------------------------------------------

// TableName 设置图片存储表的表名
func (f *FileInfo) TableName() string {
	return config.FileTableName
}

// StoragePath 获取文件的保存路径
func (f *FileInfo) StoragePath() string {
	var storage string
	switch f.FileType {
	case "jpeg", "jpg", "png", "webp":
		storage = strings.Replace(f.Link, config.FilmPictureAccess, fmt.Sprint(config.FilmPictureUploadDir, "/"), 1)
	default:
	}
	return storage
}

// CreateFileTable 创建图片关联信息存储表
func CreateFileTable() {
	// 如果不存在则创建表 并设置自增ID初始值为10000
	if !ExistFileTable() {
		err := db.Mdb.AutoMigrate(&FileInfo{})
		if err != nil {
			log.Println("Create Table FileInfo Failed: ", err)
		}
	}
}

// ExistFileTable 是否存在Picture表
func ExistFileTable() bool {
	// 1. 判断表中是否存在当前表
	return db.Mdb.Migrator().HasTable(&FileInfo{})
}

// SaveGallery 保存图片关联信息
func SaveGallery(f FileInfo) {
	db.Mdb.Create(&f)
}

// ExistFileInfoByRid 查找图片信息是否存在
// db.Mdb == nil 视为"无关联", 避免在尚未初始化 mysql 的运行环境(单元测试等)触发空指针.
func ExistFileInfoByRid(rid int64) bool {
	if db.Mdb == nil {
		return false
	}
	var count int64
	db.Mdb.Model(&FileInfo{}).Where("relevance_id = ?", rid).Count(&count)
	return count > 0
}

// GetFileInfoByRid 通过关联的资源id获取对应的图片信息.
// 找不到记录或 db.Mdb 未初始化时返回零值 (调用方应判断 f.ID == 0).
func GetFileInfoByRid(rid int64) FileInfo {
	var f FileInfo
	if db.Mdb == nil {
		return f
	}
	db.Mdb.Where("relevance_id = ?", rid).First(&f)
	return f
}

// FillBasicInfoPics 批量替换 MovieBasicInfo 列表中的封面图为本地图片.
// 替代逐条 ReplaceBasicDetailPic (Count + First 两次 SQL × N), 改为
// 一次 SELECT relevance_id IN(...) 拿回全部记录, 再用内存 map 做替换.
// 一页 14 部影片由 28 次 SQL 缩减到 1 次.
//
// 限定 type = 0 (影片封面), 防止将来 FileInfo 复用做其他类型关联时拿到错误记录.
func FillBasicInfoPics(list []MovieBasicInfo) {
	if len(list) == 0 || db.Mdb == nil {
		return
	}
	ids := make([]int64, 0, len(list))
	for _, b := range list {
		ids = append(ids, b.Id)
	}
	var files []FileInfo
	if err := db.Mdb.Where("relevance_id IN ? AND type = ?", ids, 0).Find(&files).Error; err != nil {
		log.Printf("FillBasicInfoPics query err: %v", err)
		return
	}
	if len(files) == 0 {
		return
	}
	m := make(map[int64]string, len(files))
	for _, f := range files {
		m[f.RelevanceId] = f.Link
	}
	for i := range list {
		if link, ok := m[list[i].Id]; ok {
			list[i].Picture = link
		}
	}
}

// GetFileInfoById 通过ID获取对应的图片信息
func GetFileInfoById(id uint) FileInfo {
	var f = FileInfo{}
	db.Mdb.First(&f, id)
	return f
}

// GetFileInfoPage 获取文件关联信息分页数据
func GetFileInfoPage(tl []string, page *Page) []FileInfo {
	var fl []FileInfo
	query := db.Mdb.Model(&FileInfo{}).Where("file_type IN ?", tl).Order("id DESC")
	// 获取分页相关参数
	GetPage(query, page)
	// 获取分页数据
	if err := query.Limit(page.PageSize).Offset((page.Current - 1) * page.PageSize).Find(&fl).Error; err != nil {
		log.Println(err)
		return nil
	}
	return fl
}

func DelFileInfo(id uint) {
	db.Mdb.Unscoped().Delete(&FileInfo{}, id)
}

//------------------------------------------------图片同步------------------------------------------------

// SaveVirtualPic 保存待同步的图片信息
func SaveVirtualPic(pl []VirtualPicture) error {
	// 保存对应的待同步图片信息
	var zl []redis.Z
	for _, p := range pl {
		// 首先查询 Gallery 表中是否存在当前ID对应的图片信息, 如果不存在则保存
		//if !ExistPictureByRid(p.Id) {
		//	m, _ := json.Marshal(p)
		//	zl = append(zl, redis.Z{Score: float64(p.Id), Member: m})
		//}

		// 只要开启图片同步则将图片信息存入待同步图片信息集合中, 是否同步图片交由真正同步到本地时进行决断
		m, _ := json.Marshal(p)
		zl = append(zl, redis.Z{Score: float64(p.Id), Member: m})
	}
	return db.Rdb.ZAdd(db.Cxt, config.VirtualPictureKey, zl...).Err()
}

// SyncFilmPicture 同步新采集入栈还未同步的图片.
// 历史实现是递归调用直到 ZSet 为空, 单次同步图片量大时存在栈溢出风险;
// 改为 for 循环, 直到 ZPopMax 返回空批次再退出.
// regex 提前编译为包级变量, 不再循环内 MustCompile.
func SyncFilmPicture() {
	for {
		sl := db.Rdb.ZPopMax(db.Cxt, config.VirtualPictureKey, config.MaxScanCount).Val()
		if len(sl) <= 0 {
			return
		}
		for _, s := range sl {
			member, ok := s.Member.(string)
			if !ok {
				continue
			}
			vp := VirtualPicture{}
			if err := json.Unmarshal([]byte(member), &vp); err != nil {
				continue
			}
			// 已经同步过则跳过
			if ExistFileInfoByRid(vp.Id) {
				continue
			}
			fileName, err := util.SaveOnlineFile(vp.Link, config.FilmPictureUploadDir)
			if err != nil {
				continue
			}
			SaveGallery(FileInfo{
				Link:        fmt.Sprint(config.FilmPictureAccess, fileName),
				Uid:         config.UserIdInitialVal,
				RelevanceId: vp.Id,
				Type:        0,
				Fid:         reFilePicExt.ReplaceAllString(fileName, ""),
				FileType:    strings.TrimPrefix(filepath.Ext(fileName), "."),
			})
		}
	}
}

// ReplaceDetailPic 将影片详情中的图片地址替换为自己的.
// 历史实现先 ExistFileInfoByRid (COUNT) 再 GetFileInfoByRid (First) 两次 SQL 查同一条;
// 现合并为一次 First + 判零, SQL 数减半.
func ReplaceDetailPic(d *MovieDetail) {
	f := GetFileInfoByRid(d.Id)
	if f.ID == 0 {
		return
	}
	d.Picture = f.Link
}

// ReplaceBasicDetailPic 替换影片基本数据中的封面图为本地图片.
// 单条调用; 列表场景请使用 FillBasicInfoPics 走批量查询.
func ReplaceBasicDetailPic(d *MovieBasicInfo) {
	f := GetFileInfoByRid(d.Id)
	if f.ID == 0 {
		return
	}
	d.Picture = f.Link
}
