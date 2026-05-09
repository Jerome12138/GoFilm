package system

import (
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"hash/fnv"
	"log"
	"regexp"
	"server/config"
	"server/plugin/db"
	"strconv"
	"strings"
	"time"
)

// Movie 影片基本信息
type Movie struct {
	Id       int64  `json:"id"`       // 影片ID
	Name     string `json:"name"`     // 影片名
	Cid      int64  `json:"cid"`      // 所属分类ID
	CName    string `json:"CName"`    // 所属分类名称
	EnName   string `json:"enName"`   // 英文片名
	Time     string `json:"time"`     // 更新时间
	Remarks  string `json:"remarks"`  // 备注 | 清晰度
	PlayFrom string `json:"playFrom"` // 播放来源
}

// MovieDescriptor 影片详情介绍信息
type MovieDescriptor struct {
	SubTitle    string `json:"subTitle"`    //子标题
	CName       string `json:"cName"`       //分类名称
	EnName      string `json:"enName"`      //英文名
	Initial     string `json:"initial"`     //首字母
	ClassTag    string `json:"classTag"`    //分类标签
	Actor       string `json:"actor"`       //主演
	Director    string `json:"director"`    //导演
	Writer      string `json:"writer"`      //作者
	Blurb       string `json:"blurb"`       //简介, 残缺,不建议使用
	Remarks     string `json:"remarks"`     // 更新情况
	ReleaseDate string `json:"releaseDate"` //上映时间
	Area        string `json:"area"`        // 地区
	Language    string `json:"language"`    //语言
	Year        string `json:"year"`        //年份
	State       string `json:"state"`       //影片状态 正片|预告...
	UpdateTime  string `json:"updateTime"`  //更新时间
	AddTime     int64  `json:"addTime"`     //资源添加时间戳
	DbId        int64  `json:"dbId"`        //豆瓣id
	DbScore     string `json:"dbScore"`     // 豆瓣评分
	Hits        int64  `json:"hits"`        //影片热度
	Content     string `json:"content"`     //内容简介
}

// MovieBasicInfo 影片基本信息
type MovieBasicInfo struct {
	Id       int64  `json:"id"`       //影片Id
	Cid      int64  `json:"cid"`      //分类ID
	Pid      int64  `json:"pid"`      //一级分类ID
	Name     string `json:"name"`     //片名
	SubTitle string `json:"subTitle"` //子标题
	CName    string `json:"cName"`    //分类名称
	State    string `json:"state"`    //影片状态 正片|预告...
	Picture  string `json:"picture"`  //简介图片
	Actor    string `json:"actor"`    //主演
	Director string `json:"director"` //导演
	Blurb    string `json:"blurb"`    //简介, 不完整
	Remarks  string `json:"remarks"`  // 更新情况
	Area     string `json:"area"`     // 地区
	Year     string `json:"year"`     //年份
}

// MovieUrlInfo 影视资源url信息
type MovieUrlInfo struct {
	Episode string `json:"episode"` // 集数
	Link    string `json:"link"`    // 播放地址
}

// 热路径正则统一一次性编译, 避免每次调用重复 MustCompile
var (
	reHashSpace  = regexp.MustCompile(`\s`)
	reHashAlias  = regexp.MustCompile(`～.*～$`)
	reHashPunct  = regexp.MustCompile(`^[[:punct:]]+|[[:punct:]]+$`)
	reHashSeason = regexp.MustCompile(`季.*`)
	reYear4      = regexp.MustCompile(`[1-9][0-9]{3}`)
)

// MovieDetail 影片详情信息
type MovieDetail struct {
	Id       int64    `json:"id"`       //影片Id
	Cid      int64    `json:"cid"`      //分类ID
	Pid      int64    `json:"pid"`      //一级分类ID
	Name     string   `json:"name"`     //片名
	Picture  string   `json:"picture"`  //简介图片
	PlayFrom []string `json:"playFrom"` // 播放来源
	DownFrom string   `json:"DownFrom"` //下载来源 例: http
	//PlaySeparator   string              `json:"playSeparator"` // 播放信息分隔符
	PlayList        [][]MovieUrlInfo    `json:"playList"`     //播放地址url
	DownloadList    [][]MovieUrlInfo    `json:"downloadList"` // 下载url地址
	MovieDescriptor `json:"descriptor"` //影片描述信息
}

// ===================================Redis数据交互========================================================

// SaveDetails 保存影片详情信息到redis中 格式: MovieDetail:Cid?:Id?
// detail / basicInfo 用 pipeline 打包写入避免 N 次 RTT;
// search tag 走批量 BatchHandleSearchTag (内存聚合 + pipeline);
// SearchInfo 列表只转换一次, 同时供 tag 写入和 ZSet 暂存使用.
func SaveDetails(list []MovieDetail) error {
	if len(list) == 0 {
		return nil
	}
	pipe := db.Rdb.Pipeline()
	infos := make([]SearchInfo, 0, len(list))
	for _, detail := range list {
		if data, e := json.Marshal(detail); e == nil {
			pipe.Set(db.Cxt, fmt.Sprintf(config.MovieDetailKey, detail.Cid, detail.Id), data, config.CategoryTreeExpired)
		}
		basic := buildMovieBasicInfo(detail)
		if data, e := json.Marshal(basic); e == nil {
			pipe.Set(db.Cxt, fmt.Sprintf(config.MovieBasicInfoKey, detail.Cid, detail.Id), data, config.CategoryTreeExpired)
		}
		infos = append(infos, ConvertSearchInfo(detail))
	}
	if _, err := pipe.Exec(db.Cxt); err != nil {
		log.Printf("SaveDetails pipeline exec err: %v", err)
		return err
	}
	// 批量写 search tag (pipeline + 内存聚合, 同 (key,member) 累加合并)
	BatchHandleSearchTag(infos...)
	// 暂存到 ZSet, 后续 SyncSearchInfo 异步同步到 MySQL
	RdbSaveSearchInfo(infos)
	return nil
}

// buildMovieBasicInfo 从 MovieDetail 投影出 MovieBasicInfo (仅供 SaveDetails 复用)
func buildMovieBasicInfo(detail MovieDetail) MovieBasicInfo {
	return MovieBasicInfo{
		Id:       detail.Id,
		Cid:      detail.Cid,
		Pid:      detail.Pid,
		Name:     detail.Name,
		SubTitle: detail.SubTitle,
		CName:    detail.CName,
		State:    detail.State,
		Picture:  detail.Picture,
		Actor:    detail.Actor,
		Director: detail.Director,
		Blurb:    detail.Blurb,
		Remarks:  detail.Remarks,
		Area:     detail.Area,
		Year:     detail.Year,
	}
}

// SaveDetail 保存单部影片信息
func SaveDetail(detail MovieDetail) error {
	data, err := json.Marshal(detail)
	if err != nil {
		return fmt.Errorf("marshal detail: %w", err)
	}
	if err := db.Rdb.Set(db.Cxt, fmt.Sprintf(config.MovieDetailKey, detail.Cid, detail.Id), data, config.CategoryTreeExpired).Err(); err != nil {
		return err
	}
	SaveMovieBasicInfo(detail)
	searchInfo := ConvertSearchInfo(detail)
	SaveSearchTag(searchInfo)
	return SaveSearchInfo(searchInfo)
}

// SaveMovieBasicInfo 摘取影片的详情部分信息转存为影视基本信息
func SaveMovieBasicInfo(detail MovieDetail) {
	basicInfo := MovieBasicInfo{
		Id:       detail.Id,
		Cid:      detail.Cid,
		Pid:      detail.Pid,
		Name:     detail.Name,
		SubTitle: detail.SubTitle,
		CName:    detail.CName,
		State:    detail.State,
		Picture:  detail.Picture,
		Actor:    detail.Actor,
		Director: detail.Director,
		Blurb:    detail.Blurb,
		Remarks:  detail.Remarks,
		Area:     detail.Area,
		Year:     detail.Year,
	}
	data, _ := json.Marshal(basicInfo)
	_ = db.Rdb.Set(db.Cxt, fmt.Sprintf(config.MovieBasicInfoKey, detail.Cid, detail.Id), data, config.CategoryTreeExpired).Err()
}

// SaveSitePlayList 仅保存播放url列表信息到当前站点
func SaveSitePlayList(id string, list []MovieDetail) (err error) {
	// 如果list 为空则直接返回
	if len(list) <= 0 {
		return nil
	}
	res := make(map[string]string)
	for _, d := range list {
		if len(d.PlayList) > 0 {
			data, _ := json.Marshal(d.PlayList[0])
			// 不保存电影解说类
			if strings.Contains(d.CName, "解说") {
				continue
			}
			// 如果DbId不为0, 则以dbID作为key进行hash额外存储一次
			if d.DbId != 0 {
				res[GenerateHashKey(d.DbId)] = string(data)
			}
			res[GenerateHashKey(d.Name)] = string(data)
		}
	}
	// 如果结果不为空,则将数据保存到redis中
	if len(res) > 0 {
		// 保存形式 key: MultipleSource:siteName Hash[hash(movieName)]list
		err = db.Rdb.HMSet(db.Cxt, fmt.Sprintf(config.MultipleSiteDetail, id), res).Err()
	}
	return
}

// ConvertSearchInfo 将detail信息处理成 searchInfo
func ConvertSearchInfo(detail MovieDetail) SearchInfo {
	score, _ := strconv.ParseFloat(detail.DbScore, 64)
	stamp, _ := time.ParseInLocation(time.DateTime, detail.UpdateTime, time.Local)
	// detail中的年份信息并不准确, 因此采用 ReleaseDate中的年份
	year, err := strconv.ParseInt(reYear4.FindString(detail.ReleaseDate), 10, 64)
	if err != nil {
		year = 0
	}
	return SearchInfo{
		Mid:         detail.Id,
		Cid:         detail.Cid,
		Pid:         detail.Pid,
		Name:        detail.Name,
		SubTitle:    detail.SubTitle,
		CName:       detail.CName,
		ClassTag:    detail.ClassTag,
		Area:        detail.Area,
		Language:    detail.Language,
		Year:        year,
		Initial:     detail.Initial,
		Score:       score,
		Hits:        detail.Hits,
		UpdateStamp: stamp.Unix(),
		State:       detail.State,
		Remarks:     detail.Remarks,
		// ReleaseDate 部分影片缺失该参数, 所以使用添加时间作为上映时间排序
		ReleaseStamp: detail.AddTime,
	}
}

// GetDetailByKey 获取影片对应的详情信息.
// key 不存在时 redis Get 返回空串, 此处按 zero value 返回, 调用方应判断 detail.Id == 0.
func GetDetailByKey(key string) MovieDetail {
	data, err := db.Rdb.Get(db.Cxt, key).Result()
	if err == redis.Nil {
		return MovieDetail{}
	}
	if err != nil {
		log.Printf("GetDetailByKey %s err: %v", key, err)
		return MovieDetail{}
	}
	detail := MovieDetail{}
	if err := json.Unmarshal([]byte(data), &detail); err != nil {
		log.Printf("GetDetailByKey unmarshal %s err: %v", key, err)
		return MovieDetail{}
	}
	ReplaceDetailPic(&detail)
	return detail
}

// GetBasicInfoBySearchInfos 通过searchInfo 获取影片的基本信息
// 改用 MGET 一次拿回所有 basicInfo, 避免 N 次 RTT
func GetBasicInfoBySearchInfos(infos ...SearchInfo) []MovieBasicInfo {
	if len(infos) == 0 {
		return nil
	}
	keys := make([]string, len(infos))
	for i, s := range infos {
		keys[i] = fmt.Sprintf(config.MovieBasicInfoKey, s.Cid, s.Mid)
	}
	return mgetBasicInfo(keys)
}

// mgetBasicInfo 给定 redis keys, 用 MGET 批量拉回 MovieBasicInfo, 顺序与入参一致
func mgetBasicInfo(keys []string) []MovieBasicInfo {
	if len(keys) == 0 {
		return nil
	}
	vals, err := db.Rdb.MGet(db.Cxt, keys...).Result()
	if err != nil {
		log.Printf("mgetBasicInfo err: %v", err)
		return nil
	}
	list := make([]MovieBasicInfo, 0, len(vals))
	for _, v := range vals {
		if v == nil {
			continue
		}
		s, ok := v.(string)
		if !ok || len(s) == 0 {
			continue
		}
		basic := MovieBasicInfo{}
		if err := json.Unmarshal([]byte(s), &basic); err != nil {
			continue
		}
		ReplaceBasicDetailPic(&basic)
		list = append(list, basic)
	}
	return list
}

/*
	对附属播放源入库时的name|dbID进行处理,保证唯一性
1. 去除name中的所有空格
2. 去除name中含有的别名～.*～
3. 去除name首尾的标点符号
4. 将处理完成后的name转化为hash值作为存储时的key
*/
// GenerateHashKey 存储播放源信息时对影片名称进行处理, 提高各站点间同一影片的匹配度
func GenerateHashKey[K string | ~int | int64](key K) string {
	mName := fmt.Sprint(key)
	mName = reHashSpace.ReplaceAllString(mName, "")
	mName = reHashAlias.ReplaceAllString(mName, "")
	mName = reHashPunct.ReplaceAllString(mName, "")
	mName = reHashSeason.ReplaceAllString(mName, "季")
	h := fnv.New32a()
	_, _ = h.Write([]byte(mName))
	return fmt.Sprint(h.Sum32())
}

// ============================采集方案.v1 遗留==================================================

// SaveMoves  保存影片分页请求list
func SaveMoves(list []Movie) (err error) {
	// 整合数据
	for _, m := range list {
		//score, _ := time.ParseInLocation(time.DateTime, m.Time, time.Local)
		movie, _ := json.Marshal(m)
		// 以Cid为目录为集合进行存储, 便于后续搜索, 以影片id为分值进行存储 例 MovieList:Cid%d
		err = db.Rdb.ZAdd(db.Cxt, fmt.Sprintf(config.MovieListInfoKey, m.Cid), redis.Z{Score: float64(m.Id), Member: movie}).Err()
	}
	return err
}

// AllMovieInfoKey 获取redis中所有的影视列表信息key MovieList:Cid
func AllMovieInfoKey() []string {
	return db.Rdb.Keys(db.Cxt, fmt.Sprint("MovieList:Cid*")).Val()
}

// GetMovieListByKey 获取指定分类的影片列表数据
func GetMovieListByKey(key string) []string {
	return db.Rdb.ZRange(db.Cxt, key, 0, -1).Val()
}
