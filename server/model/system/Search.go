package system

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log"
	"math"
	"reflect"
	"regexp"
	"server/config"
	"server/plugin/common/param"
	"server/plugin/db"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 热路径正则一次性编译
var (
	reTagWhitespace = regexp.MustCompile(`[\s\n\r]+`)
	reRelateName    = regexp.MustCompile(`(第.{1,3}季.*)|([0-9]{1,3})|(剧场版)|(\s\S*$)|(之.*)|([\p{P}\p{S}].*)`)
)

// searchTagInitialized 标记某个 (pid, fieldKey) 的固定 tag 已经写入过 redis,
// 避免每次保存影片都重复 ZCard 探测同一个 key 是否为空.
var (
	searchTagInitMu  sync.RWMutex
	searchTagInitMap = make(map[string]struct{})
)

func searchTagWasInit(key string) bool {
	searchTagInitMu.RLock()
	_, ok := searchTagInitMap[key]
	searchTagInitMu.RUnlock()
	return ok
}

func searchTagMarkInit(key string) {
	searchTagInitMu.Lock()
	searchTagInitMap[key] = struct{}{}
	searchTagInitMu.Unlock()
}

// SearchInfo 存储用于检索的信息
type SearchInfo struct {
	gorm.Model
	Mid          int64   `json:"mid"`          //影片ID gorm:"uniqueIndex:idx_mid"
	Cid          int64   `json:"cid"`          //分类ID
	Pid          int64   `json:"pid"`          //上级分类ID
	Name         string  `json:"name"`         // 片名
	SubTitle     string  `json:"subTitle"`     // 影片子标题
	CName        string  `json:"cName"`        // 分类名称
	ClassTag     string  `json:"classTag"`     //类型标签
	Area         string  `json:"area"`         // 地区
	Language     string  `json:"language"`     // 语言
	Year         int64   `json:"year"`         // 年份
	Initial      string  `json:"initial"`      // 首字母
	Score        float64 `json:"score"`        //评分
	UpdateStamp  int64   `json:"updateStamp"`  // 更新时间
	Hits         int64   `json:"hits"`         // 热度排行
	State        string  `json:"state"`        //状态 正片|预告
	Remarks      string  `json:"remarks"`      // 完结 | 更新至x集
	ReleaseStamp int64   `json:"releaseStamp"` //上映时间 时间戳
}

// Tag 影片分类标签结构体
type Tag struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

func (s *SearchInfo) TableName() string {
	return config.SearchTableName
}

// ================================= Spider 数据处理(redis) =================================

// RdbSaveSearchInfo 批量保存检索信息到redis
func RdbSaveSearchInfo(list []SearchInfo) {
	// 1.整合一下zset数据集
	var members []redis.Z
	for _, s := range list {
		member, _ := json.Marshal(s)
		members = append(members, redis.Z{Score: float64(s.Mid), Member: member})
	}
	// 2.批量保存到zset集合中
	db.Rdb.ZAdd(db.Cxt, config.SearchInfoTemp, members...)
}

// FilmZero 删除所有库存数据
// 使用 SCAN 替代 KEYS, 防止大库扫描阻塞 redis 主线程; 删除时按批 Unlink 避免单次删除耗时长
func FilmZero() {
	for _, pattern := range []string{
		"MovieBasicInfo:*",
		"MovieDetail:*",
		"MultipleSource:*",
		"OriginalResource:*",
		"Search:*",
	} {
		scanAndDelete(pattern, 500)
	}
	// 删除mysql中留存的检索表
	var s *SearchInfo
	if ExistSearchTable() {
		db.Mdb.Exec(fmt.Sprintf(`TRUNCATE table %s`, s.TableName()))
	}
}

// scanAndDelete 使用 SCAN 清理匹配 pattern 的 key.
// 先完整收集 (SCAN 允许重复, 用 set 去重), 再分批 Unlink/Del 删除,
// 避免边扫边删导致 cursor 失效漏 key.
func scanAndDelete(pattern string, batch int64) {
	seen := make(map[string]struct{})
	var cursor uint64
	for {
		keys, nextCursor, err := db.Rdb.Scan(db.Cxt, cursor, pattern, batch).Result()
		if err != nil {
			log.Printf("scanAndDelete %s err: %v", pattern, err)
			return
		}
		for _, k := range keys {
			seen[k] = struct{}{}
		}
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}
	if len(seen) == 0 {
		return
	}
	list := make([]string, 0, len(seen))
	for k := range seen {
		list = append(list, k)
	}
	const chunk = 500
	for i := 0; i < len(list); i += chunk {
		end := i + chunk
		if end > len(list) {
			end = len(list)
		}
		if err := db.Rdb.Unlink(db.Cxt, list[i:end]...).Err(); err != nil {
			if delErr := db.Rdb.Del(db.Cxt, list[i:end]...).Err(); delErr != nil {
				log.Printf("scanAndDelete Del %s err: %v", pattern, delErr)
			}
		}
	}
}

// ResetSearchTable 重置Search表
func ResetSearchTable() {
	// 删除 Search 表
	var s *SearchInfo
	db.Mdb.Exec(fmt.Sprintf(`drop table if exists %s`, s.TableName()))
	// 重新创建 Search 表
	CreateSearchTable()
}

// DelMtPlay 清空附加播放源信息
func DelMtPlay(keys []string) {
	db.Rdb.Del(db.Cxt, keys...)
}

/*
SearchKeyword 设置search关键字集合(影片分类检索类型数据)
	类型, 剧情 , 地区, 语言, 年份, 首字母, 排序
	1. 在影片详情缓存到redis时将影片的相关数据进行记录, 存在相同类型则分值加一
	2. 通过分值对类型进行排序类型展示到页面
*/

// searchTitleFields 是一组固定的 title 维度, 不再每次都 HGetAll 探测
var searchTitleFields = map[string]string{
	"Category": "类型",
	"Plot":     "剧情",
	"Area":     "地区",
	"Language": "语言",
	"Year":     "年份",
	"Initial":  "首字母",
	"Sort":     "排序",
}

func SaveSearchTag(search SearchInfo) {
	titleKey := fmt.Sprintf(config.SearchTitle, search.Pid)
	// title 字段集只在该 pid 第一次出现时落盘一次
	if !searchTagWasInit(titleKey) {
		if exists, _ := db.Rdb.Exists(db.Cxt, titleKey).Result(); exists == 0 {
			db.Rdb.HMSet(db.Cxt, titleKey, searchTitleFields)
		}
		searchTagMarkInit(titleKey)
	}
	// 对每个固定字段执行对应处理: 静态枚举的字段一次性落 redis, 动态字段累加
	for k := range searchTitleFields {
		tagKey := fmt.Sprintf(config.SearchTag, search.Pid, k)
		switch k {
		case "Category":
			ensureCategoryTags(search.Pid, tagKey)
		case "Year":
			ensureYearTags(tagKey)
		case "Initial":
			ensureInitialTags(tagKey)
		case "Sort":
			ensureSortTags(tagKey)
		case "Plot":
			HandleSearchTags(search.ClassTag, tagKey)
		case "Area":
			HandleSearchTags(search.Area, tagKey)
		case "Language":
			HandleSearchTags(search.Language, tagKey)
		}
	}
}

func ensureCategoryTags(pid int64, tagKey string) {
	if searchTagWasInit(tagKey) {
		return
	}
	if db.Rdb.ZCard(db.Cxt, tagKey).Val() > 0 {
		searchTagMarkInit(tagKey)
		return
	}
	for _, t := range GetChildrenTree(pid) {
		db.Rdb.ZAdd(db.Cxt, tagKey, redis.Z{Score: float64(-t.Id), Member: fmt.Sprintf("%v:%v", t.Name, t.Id)})
	}
	searchTagMarkInit(tagKey)
}

func ensureYearTags(tagKey string) {
	if searchTagWasInit(tagKey) {
		return
	}
	if db.Rdb.ZCard(db.Cxt, tagKey).Val() > 0 {
		searchTagMarkInit(tagKey)
		return
	}
	currentYear := time.Now().Year()
	members := make([]redis.Z, 0, 12)
	for i := 0; i < 12; i++ {
		members = append(members, redis.Z{Score: float64(currentYear - i), Member: fmt.Sprintf("%v:%v", currentYear-i, currentYear-i)})
	}
	db.Rdb.ZAdd(db.Cxt, tagKey, members...)
	searchTagMarkInit(tagKey)
}

func ensureInitialTags(tagKey string) {
	if searchTagWasInit(tagKey) {
		return
	}
	if db.Rdb.ZCard(db.Cxt, tagKey).Val() > 0 {
		searchTagMarkInit(tagKey)
		return
	}
	members := make([]redis.Z, 0, 26)
	for i := 65; i <= 90; i++ {
		members = append(members, redis.Z{Score: float64(90 - i), Member: fmt.Sprintf("%c:%c", i, i)})
	}
	db.Rdb.ZAdd(db.Cxt, tagKey, members...)
	searchTagMarkInit(tagKey)
}

func ensureSortTags(tagKey string) {
	if searchTagWasInit(tagKey) {
		return
	}
	if db.Rdb.ZCard(db.Cxt, tagKey).Val() > 0 {
		searchTagMarkInit(tagKey)
		return
	}
	db.Rdb.ZAdd(db.Cxt, tagKey,
		redis.Z{Score: 3, Member: "时间排序:update_stamp"},
		redis.Z{Score: 2, Member: "人气排序:hits"},
		redis.Z{Score: 1, Member: "评分排序:score"},
		redis.Z{Score: 0, Member: "最新上映:release_stamp"},
	)
	searchTagMarkInit(tagKey)
}

func HandleSearchTags(preTags string, k string) {
	// 先处理字符串中的空白符 然后对处理前的tag字符串进行分割
	preTags = reTagWhitespace.ReplaceAllString(preTags, "")
	// ZIncrBy 一步完成 读 + 累加 + 写, 比原 ZScore + ZAdd 减半 RTT, 也避免并发竞态
	incr := func(member string) {
		db.Rdb.ZIncrBy(db.Cxt, k, 1, member)
	}
	splitIncr := func(sep string) {
		for _, t := range strings.Split(preTags, sep) {
			incr(fmt.Sprintf("%v:%v", t, t))
		}
	}
	switch {
	case strings.Contains(preTags, "/"):
		splitIncr("/")
	case strings.Contains(preTags, ","):
		splitIncr(",")
	case strings.Contains(preTags, "，"):
		splitIncr("，")
	case strings.Contains(preTags, "、"):
		splitIncr("、")
	default:
		if len(preTags) == 0 {
			return
		}
		if preTags == "其它" {
			// "其它" 仅作为占位 tag, 保持原 score=0 行为
			db.Rdb.ZAdd(db.Cxt, k, redis.Z{Score: 0, Member: fmt.Sprintf("%v:%v", preTags, preTags)})
			return
		}
		incr(fmt.Sprintf("%v:%v", preTags, preTags))
	}
}

// BatchHandleSearchTag 批量写入 search tag.
// 相比 for-loop 调用 SaveSearchTag, 主要做两件事:
//  1. 同一 pid 的静态 tag (Year/Initial/Sort/Category 以及 title HMSet) 仅 init 一次;
//  2. 动态 tag (Plot/Area/Language) 在内存聚合 (tagKey, member) 增量, 用 pipeline 一次发出 ZIncrBy.
//     原实现每部影片每个 tag 都是独立 ZIncrBy, 30 部 ×3 维度 ≈ 上百次 RTT.
func BatchHandleSearchTag(infos ...SearchInfo) {
	if len(infos) == 0 {
		return
	}
	// 1. 每个 pid 的静态 tag 仅 init 一次
	seenPid := make(map[int64]struct{})
	for _, info := range infos {
		if _, ok := seenPid[info.Pid]; ok {
			continue
		}
		seenPid[info.Pid] = struct{}{}
		ensureStaticTagsForPid(info.Pid)
	}
	// 2. 动态 tag 聚合, pipeline 一次性发出
	incs := make(map[string]map[string]int64)
	placeholders := make(map[string]map[string]struct{})
	for _, info := range infos {
		aggregateDynamicTag(incs, placeholders, fmt.Sprintf(config.SearchTag, info.Pid, "Plot"), info.ClassTag)
		aggregateDynamicTag(incs, placeholders, fmt.Sprintf(config.SearchTag, info.Pid, "Area"), info.Area)
		aggregateDynamicTag(incs, placeholders, fmt.Sprintf(config.SearchTag, info.Pid, "Language"), info.Language)
	}
	if len(incs) == 0 && len(placeholders) == 0 {
		return
	}
	pipe := db.Rdb.Pipeline()
	for k, m := range incs {
		for member, n := range m {
			pipe.ZIncrBy(db.Cxt, k, float64(n), member)
		}
	}
	for k, members := range placeholders {
		for m := range members {
			pipe.ZAdd(db.Cxt, k, redis.Z{Score: 0, Member: m})
		}
	}
	if _, err := pipe.Exec(db.Cxt); err != nil {
		log.Printf("BatchHandleSearchTag pipeline err: %v", err)
	}
}

// ensureStaticTagsForPid 把同一 pid 下静态 tag 的 init 逻辑收敛到一次调用.
// 内部仍复用 ensureXxxTags, 因此 init 标记仍由 searchTagInitMap 保证幂等.
func ensureStaticTagsForPid(pid int64) {
	titleKey := fmt.Sprintf(config.SearchTitle, pid)
	if !searchTagWasInit(titleKey) {
		if exists, _ := db.Rdb.Exists(db.Cxt, titleKey).Result(); exists == 0 {
			db.Rdb.HMSet(db.Cxt, titleKey, searchTitleFields)
		}
		searchTagMarkInit(titleKey)
	}
	ensureCategoryTags(pid, fmt.Sprintf(config.SearchTag, pid, "Category"))
	ensureYearTags(fmt.Sprintf(config.SearchTag, pid, "Year"))
	ensureInitialTags(fmt.Sprintf(config.SearchTag, pid, "Initial"))
	ensureSortTags(fmt.Sprintf(config.SearchTag, pid, "Sort"))
}

// aggregateDynamicTag 复刻 HandleSearchTags 的 split / "其它" 占位 / 单值逻辑,
// 但只把增量记到内存 map, 不直接发 redis 命令.
func aggregateDynamicTag(incs map[string]map[string]int64, ph map[string]map[string]struct{}, tagKey, preTags string) {
	preTags = reTagWhitespace.ReplaceAllString(preTags, "")
	add := func(member string) {
		m, ok := incs[tagKey]
		if !ok {
			m = make(map[string]int64)
			incs[tagKey] = m
		}
		m[member]++
	}
	splitAdd := func(sep string) {
		for _, t := range strings.Split(preTags, sep) {
			add(fmt.Sprintf("%v:%v", t, t))
		}
	}
	switch {
	case strings.Contains(preTags, "/"):
		splitAdd("/")
	case strings.Contains(preTags, ","):
		splitAdd(",")
	case strings.Contains(preTags, "，"):
		splitAdd("，")
	case strings.Contains(preTags, "、"):
		splitAdd("、")
	default:
		if len(preTags) == 0 {
			return
		}
		if preTags == "其它" {
			m, ok := ph[tagKey]
			if !ok {
				m = make(map[string]struct{})
				ph[tagKey] = m
			}
			m[fmt.Sprintf("%v:%v", preTags, preTags)] = struct{}{}
			return
		}
		add(fmt.Sprintf("%v:%v", preTags, preTags))
	}
}

// ================================= Spider 数据处理(mysql) =================================

// CreateSearchTable 创建存储检索信息的数据表
func CreateSearchTable() {
	// 如果不存在则创建表
	if !ExistSearchTable() {
		err := db.Mdb.AutoMigrate(&SearchInfo{})
		if err != nil {
			log.Println("Create Table SearchInfo Failed: ", err)
		}
	}
}

func ExistSearchTable() bool {
	// 1. 判断表中是否存在当前表
	return db.Mdb.Migrator().HasTable(&SearchInfo{})
}

// AddSearchIndex search表中数据保存完毕后 将常用字段添加索引提高查询效率
func AddSearchIndex() {
	var s *SearchInfo
	tableName := s.TableName()
	// 添加索引
	db.Mdb.Exec(fmt.Sprintf("CREATE UNIQUE INDEX idx_mid ON %s (mid)", tableName))
	db.Mdb.Exec(fmt.Sprintf("CREATE INDEX idx_time ON %s (update_stamp DESC)", tableName))
	db.Mdb.Exec(fmt.Sprintf("CREATE INDEX idx_hits ON %s (hits DESC)", tableName))
	db.Mdb.Exec(fmt.Sprintf("CREATE INDEX idx_score ON %s (score DESC)", tableName))
	db.Mdb.Exec(fmt.Sprintf("CREATE INDEX idx_release ON %s (release_stamp DESC)", tableName))
	db.Mdb.Exec(fmt.Sprintf("CREATE INDEX idx_year ON %s (year DESC)", tableName))

}

// searchInsertBatchSize CreateInBatches 单次 INSERT 的行数上限.
// 太小会增加事务往返, 太大会让 packet/log 体积剧增 (mysql max_allowed_packet 默认 64MB).
const searchInsertBatchSize = 200

// BatchSave 批量保存影片search信息
func BatchSave(list []SearchInfo) {
	if len(list) == 0 {
		return
	}
	tx := db.Mdb.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("BatchSave panic recovered: %v", r)
		}
	}()
	if err := tx.CreateInBatches(list, searchInsertBatchSize).Error; err != nil {
		// 插入失败回滚事务, 不再 commit, 也不应将"未持久化"的 tag 写入 redis
		tx.Rollback()
		log.Printf("BatchSave CreateInBatches err: %v", err)
		return
	}
	if err := tx.Commit().Error; err != nil {
		log.Printf("BatchSave Commit err: %v", err)
		return
	}
	// 仅在事务提交成功后再写 redis tag, 保持双侧一致
	BatchHandleSearchTag(list...)
}

// BatchSaveOrUpdate 批量 upsert 影片 search 信息.
// 历史实现对每条记录走 SELECT COUNT + UPDATE/INSERT 双 SQL, 300 条 = 600 SQL 严重制约同步速度.
// 现改为:
//  1. 一次 SELECT mid IN(...) 拉回已存在 mid 集合, 用于区分 "新增 vs 更新" 以决定是否累加 redis tag;
//  2. 用 ON DUPLICATE KEY UPDATE (依赖 unique idx_mid) 一次 batch upsert, 命中已存在记录即按指定列覆盖.
//
// 仅对新增记录写 redis tag, 与历史语义一致 (避免重复累加 tag 计数).
func BatchSaveOrUpdate(list []SearchInfo) {
	if len(list) == 0 {
		return
	}
	tx := db.Mdb.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("BatchSaveOrUpdate panic recovered: %v", r)
		}
	}()
	mids := make([]int64, 0, len(list))
	for _, info := range list {
		mids = append(mids, info.Mid)
	}
	var existMids []int64
	if err := tx.Model(&SearchInfo{}).Where("mid IN ?", mids).Pluck("mid", &existMids).Error; err != nil {
		tx.Rollback()
		log.Printf("BatchSaveOrUpdate pluck existing mid err: %v", err)
		return
	}
	existSet := make(map[int64]struct{}, len(existMids))
	for _, m := range existMids {
		existSet[m] = struct{}{}
	}
	// upsert: 已存在则覆盖指定列, 不存在则插入
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "mid"}},
		DoUpdates: clause.AssignmentColumns([]string{"update_stamp", "hits", "state", "remarks", "score", "release_stamp"}),
	}).CreateInBatches(list, searchInsertBatchSize).Error; err != nil {
		tx.Rollback()
		log.Printf("BatchSaveOrUpdate upsert err: %v", err)
		return
	}
	if err := tx.Commit().Error; err != nil {
		log.Printf("BatchSaveOrUpdate commit err: %v", err)
		return
	}
	// 只对新增的记录累加 redis tag, 维持原"更新不累加"的语义
	inserted := make([]SearchInfo, 0, len(list)-len(existSet))
	for _, info := range list {
		if _, ok := existSet[info.Mid]; !ok {
			inserted = append(inserted, info)
		}
	}
	BatchHandleSearchTag(inserted...)
}

// SaveSearchInfo 添加影片检索信息
func SaveSearchInfo(s SearchInfo) error {
	// 先查询数据库中是否存在对应记录
	// 如果不存在对应记录则 保存当前记录
	tx := db.Mdb.Begin()
	if !ExistSearchInfo(s.Mid) {
		// 执行插入操作
		if err := tx.Create(&s).Error; err != nil {
			tx.Rollback()
			return err
		}
		// 执行添加操作时保存一份tag信息
		BatchHandleSearchTag(s)
	} else {
		// 如果已经存在当前记录则将当前记录进行更新
		err := tx.Model(&SearchInfo{}).Where("mid", s.Mid).Updates(SearchInfo{UpdateStamp: s.UpdateStamp, Hits: s.Hits, State: s.State,
			Remarks: s.Remarks, Score: s.Score, ReleaseStamp: s.ReleaseStamp}).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	// 提交事务
	tx.Commit()
	return nil
}

// ExistSearchInfo 通过Mid查询是否存在影片的检索信息
func ExistSearchInfo(mid int64) bool {
	var count int64
	db.Mdb.Model(&SearchInfo{}).Where("mid", mid).Count(&count)
	return count > 0
}

// TunCateSearchTable 截断SearchInfo数据表
func TunCateSearchTable() {
	var searchInfo *SearchInfo
	err := db.Mdb.Exec(fmt.Sprint("TRUNCATE TABLE ", searchInfo.TableName())).Error
	if err != nil {
		log.Println("TRUNCATE TABLE Error: ", err)
	}
}

// SyncSearchInfo 同步影片检索信息
func SyncSearchInfo(model int) {
	switch model {
	case 0:
		// 重置Search表, (恢复为初始状态, 未添加索引)
		ResetSearchTable()
		// 批量添加 SearchInfo
		SearchInfoToMdb(model)
		// 保存完所有 SearchInfo 后添加字段索引
		AddSearchIndex()
	case 1:
		// 批量更新或添加
		SearchInfoToMdb(model)
	}
}

// SearchInfoToMdb 扫描redis中的检索信息, 并批量存入mysql (model 执行模式 0-清空并保存 || 1-更新)
// 改为 for 循环, 避免数据量大时递归栈溢出
func SearchInfoToMdb(model int) {
	for {
		// 1. 从 redis 中批量弹出
		list := db.Rdb.ZPopMax(db.Cxt, config.SearchInfoTemp, config.MaxScanCount).Val()
		if len(list) <= 0 {
			return
		}
		// 2. 解析
		sl := make([]SearchInfo, 0, len(list))
		for _, s := range list {
			info := SearchInfo{}
			member, ok := s.Member.(string)
			if !ok {
				continue
			}
			if err := json.Unmarshal([]byte(member), &info); err != nil {
				log.Printf("SearchInfoToMdb unmarshal err: %v", err)
				continue
			}
			sl = append(sl, info)
		}
		if len(sl) == 0 {
			continue
		}
		// 3. 持久化
		switch model {
		case 0:
			BatchSave(sl)
		case 1:
			BatchSaveOrUpdate(sl)
		}
	}
}

// ================================= API 数据接口信息处理 =================================

// GetMovieListByPid  通过Pid 分类ID 获取对应影片的数据信息
func GetMovieListByPid(pid int64, page *Page) []MovieBasicInfo {
	// 返回分页参数
	var count int64
	db.Mdb.Model(&SearchInfo{}).Where("pid", pid).Count(&count)
	page.Total = int(count)
	page.PageCount = int((page.Total + page.PageSize - 1) / page.PageSize)
	// 进行具体的信息查询
	var s []SearchInfo
	if err := db.Mdb.Limit(page.PageSize).Offset((page.Current-1)*page.PageSize).Where("pid", pid).Order("year DESC, update_stamp DESC").Find(&s).Error; err != nil {
		log.Println(err)
		return nil
	}
	return GetBasicInfoBySearchInfos(s...)
}

// GetMovieListByCid 通过Cid查找对应的影片分页数据
func GetMovieListByCid(cid int64, page *Page) []MovieBasicInfo {
	var count int64
	db.Mdb.Model(&SearchInfo{}).Where("cid", cid).Count(&count)
	page.Total = int(count)
	page.PageCount = int((page.Total + page.PageSize - 1) / page.PageSize)
	var s []SearchInfo
	if err := db.Mdb.Limit(page.PageSize).Offset((page.Current-1)*page.PageSize).Where("cid", cid).Order("year DESC, update_stamp DESC").Find(&s).Error; err != nil {
		log.Println(err)
		return nil
	}
	return GetBasicInfoBySearchInfos(s...)
}

// GetHotMovieByPid  获取指定类别的热门影片
func GetHotMovieByPid(pid int64, page *Page) []SearchInfo {
	// 返回分页参数
	var count int64
	db.Mdb.Model(&SearchInfo{}).Where("pid", pid).Count(&count)
	page.Total = int(count)
	page.PageCount = int((page.Total + page.PageSize - 1) / page.PageSize)
	// 进行具体的信息查询
	var s []SearchInfo
	// 当前时间偏移一个月
	t := time.Now().AddDate(0, -1, 0).Unix()
	if err := db.Mdb.Limit(page.PageSize).Offset((page.Current-1)*page.PageSize).Where("pid=? AND update_stamp > ?", pid, t).Order(" year DESC, hits DESC").Find(&s).Error; err != nil {
		log.Println(err)
		return nil
	}
	return s
}

// SearchFilmKeyword 通过关键字搜索库存中满足条件的影片名
func SearchFilmKeyword(keyword string, page *Page) []SearchInfo {
	var searchList []SearchInfo
	// 1. 先统计搜索满足条件的数据量
	var count int64
	db.Mdb.Model(&SearchInfo{}).Where("name LIKE ?", fmt.Sprint(`%`, keyword, `%`)).Or("sub_title LIKE ?", fmt.Sprint(`%`, keyword, `%`)).Count(&count)
	page.Total = int(count)
	page.PageCount = int((page.Total + page.PageSize - 1) / page.PageSize)
	// 2. 获取满足条件的数据
	db.Mdb.Limit(page.PageSize).Offset((page.Current-1)*page.PageSize).
		Where("name LIKE ?", fmt.Sprint(`%`, keyword, `%`)).Or("sub_title LIKE ?", fmt.Sprint(`%`, keyword, `%`)).Order("year DESC, update_stamp DESC").Find(&searchList)
	return searchList
}

// GetRelateMovieBasicInfo GetRelateMovie 根据SearchInfo获取相关影片
func GetRelateMovieBasicInfo(search SearchInfo, page *Page) []MovieBasicInfo {
	/*
		根据当前影片信息匹配相关的影片
		1. 分类Cid,
		2. 如果影片名称含有第x季 则根据影片名进行模糊匹配
		3. class_tag 剧情内容匹配, 切分后使用 or 进行匹配
		4. area 地区
		5. 语言 Language
	*/
	// sql 拼接查询条件
	sql := ""

	// 优先进行名称相似匹配
	//search.Name = regexp.MustCompile("第.{1,3}季").ReplaceAllString(search.Name, "")
	name := reRelateName.ReplaceAllString(search.Name, "")
	// 如果处理后的影片名称依旧没有改变 且具有一定长度 则截取部分内容作为搜索条件
	if len(name) == len(search.Name) && len(name) > 10 {
		// 中文字符需截取3的倍数,否则可能乱码
		name = name[:int(math.Ceil(float64(len(name)/5))*3)]
	}
	sql = fmt.Sprintf(`select * from %s where (name LIKE "%%%s%%" or sub_title LIKE "%%%[2]s%%") AND cid=%d union`, search.TableName(), name, search.Cid)
	// 执行后续匹配内容, 匹配结果过少,减少过滤条件
	//sql = fmt.Sprintf(`%s select * from %s where cid=%d AND area="%s" AND language="%s" AND`, sql, search.TableName(), search.Cid, search.Area, search.Language)

	// 添加其他相似匹配规则
	sql = fmt.Sprintf(`%s (select * from %s where cid=%d AND `, sql, search.TableName(), search.Cid)
	// 根据剧情标签查找相似影片, classTag 使用的分隔符为 , | /
	// 首先去除 classTag 中包含的所有空格
	search.ClassTag = strings.ReplaceAll(search.ClassTag, " ", "")
	// 如果 classTag 中包含分割符则进行拆分匹配
	if strings.Contains(search.ClassTag, ",") {
		s := "("
		for _, t := range strings.Split(search.ClassTag, ",") {
			s = fmt.Sprintf(`%s class_tag like "%%%s%%" OR`, s, t)
		}
		sql = fmt.Sprintf("%s %s)", sql, strings.TrimSuffix(s, "OR"))
	} else if strings.Contains(search.ClassTag, "/") {
		s := "("
		for _, t := range strings.Split(search.ClassTag, "/") {
			s = fmt.Sprintf(`%s class_tag like "%%%s%%" OR`, s, t)
		}
		sql = fmt.Sprintf("%s %s)", sql, strings.TrimSuffix(s, "OR"))
	} else {
		sql = fmt.Sprintf(`%s class_tag like "%%%s%%"`, sql, search.ClassTag)
	}
	// 除名称外的相似影片使用随机排序
	sql = fmt.Sprintf("%s ORDER BY RAND() limit %d,%d)", sql, page.Current, page.PageSize)
	// 条件拼接完成后加上limit参数
	sql = fmt.Sprintf("(%s)  limit %d,%d", sql, page.Current, page.PageSize)
	// 执行sql
	var list []SearchInfo
	db.Mdb.Raw(sql).Scan(&list)
	// 用 MGET 批量取回 basicInfo, 顺序与 list 一致
	return GetBasicInfoBySearchInfos(list...)
}

// GetMultiplePlay 通过影片名hash值匹配播放源
func GetMultiplePlay(siteId, key string) []MovieUrlInfo {
	data := db.Rdb.HGet(db.Cxt, fmt.Sprintf(config.MultipleSiteDetail, siteId), key).Val()
	var playList []MovieUrlInfo
	_ = json.Unmarshal([]byte(data), &playList)
	return playList
}

// BatchGetMultiplePlay 用 pipeline + HMGET 一次性拉回多个站点的播放源.
// 返回值切片与 sources 顺序对齐: 每项是该站点匹配到的第一个非空播放源 (空则为 nil).
// 替代 multipleSource 中 N×M 次 HGet, N 站点 × M name 时减少 RTT.
func BatchGetMultiplePlay(sources []FilmSource, keys []string) [][]MovieUrlInfo {
	result := make([][]MovieUrlInfo, len(sources))
	if len(sources) == 0 || len(keys) == 0 {
		return result
	}
	pipe := db.Rdb.Pipeline()
	cmds := make([]*redis.SliceCmd, len(sources))
	for i, s := range sources {
		cmds[i] = pipe.HMGet(db.Cxt, fmt.Sprintf(config.MultipleSiteDetail, s.Id), keys...)
	}
	if _, err := pipe.Exec(db.Cxt); err != nil && !errors.Is(err, redis.Nil) {
		log.Printf("BatchGetMultiplePlay pipeline err: %v", err)
		return result
	}
	for i, c := range cmds {
		for _, v := range c.Val() {
			if v == nil {
				continue
			}
			s, ok := v.(string)
			if !ok || len(s) == 0 {
				continue
			}
			var pl []MovieUrlInfo
			if err := json.Unmarshal([]byte(s), &pl); err == nil && len(pl) > 0 {
				result[i] = pl
				break
			}
		}
	}
	return result
}

// GetSearchTag 通过影片分类 Pid 返回对应分类的tag信息
func GetSearchTag(pid int64) map[string]interface{} {
	// 整合searchTag相关内容
	res := make(map[string]interface{})
	titles := db.Rdb.HGetAll(db.Cxt, fmt.Sprintf(config.SearchTitle, pid)).Val()
	res["titles"] = titles
	// 处理单一分类的数据格式
	tagMap := make(map[string]interface{})
	for t, _ := range titles {
		tagMap[t] = HandleTagStr(t, GetTagsByTitle(pid, t)...)
	}
	res["tags"] = tagMap
	// 分类列表展示的顺序
	res["sortList"] = []string{"Category", "Plot", "Area", "Language", "Year", "Sort"}
	return res
}

// GetTagsByTitle 返回Pid和title对应的用于检索的tag
func GetTagsByTitle(pid int64, t string) []string {
	// 通过 k 获取对应的 tag , 并以score进行排序
	var tags []string
	// 过滤分类tag
	switch t {
	case "Category":
		tags = db.Rdb.ZRevRange(db.Cxt, fmt.Sprintf(config.SearchTag, pid, t), 0, -1).Val()
	case "Plot":
		tags = db.Rdb.ZRevRange(db.Cxt, fmt.Sprintf(config.SearchTag, pid, t), 0, 10).Val()
	case "Area":
		tags = db.Rdb.ZRevRange(db.Cxt, fmt.Sprintf(config.SearchTag, pid, t), 0, 11).Val()
	case "Language":
		tags = db.Rdb.ZRevRange(db.Cxt, fmt.Sprintf(config.SearchTag, pid, t), 0, 6).Val()
	case "Year", "Initial", "Sort":
		tags = db.Rdb.ZRevRange(db.Cxt, fmt.Sprintf(config.SearchTag, pid, t), 0, -1).Val()
	default:
		break
	}
	return tags
}

// HandleTagStr 处理tag数据格式
func HandleTagStr(title string, tags ...string) []map[string]string {
	var r []map[string]string
	if !strings.EqualFold(title, "Sort") {
		r = append(r, map[string]string{
			"Name":  "全部",
			"Value": "",
		})
	}
	for _, t := range tags {
		if sl := strings.Split(t, ":"); len(sl) > 0 {
			r = append(r, map[string]string{
				"Name":  sl[0],
				"Value": sl[1],
			})
		}
	}
	if !strings.EqualFold(title, "Sort") && !strings.EqualFold(title, "Year") && !strings.EqualFold(title, "Category") {
		r = append(r, map[string]string{
			"Name":  "其它",
			"Value": "其它",
		})
	}
	return r
}

// GetSearchInfosByTags 查询满足searchTag条件的影片分页数据
func GetSearchInfosByTags(st SearchTagsVO, page *Page) []SearchInfo {
	// 准备查询语句的条件
	qw := db.Mdb.Model(&SearchInfo{})
	// 通过searchTags的非空属性值, 拼接对应的查询条件
	t := reflect.TypeOf(st)
	v := reflect.ValueOf(st)
	for i := 0; i < t.NumField(); i++ {
		// 如果字段值不为空
		value := v.Field(i).Interface()
		if !param.IsEmpty(value) {
			// 如果value是 其它 则进行特殊处理
			var ts []string
			if v, flag := value.(string); flag && strings.EqualFold(v, "其它") {
				for _, s := range GetTagsByTitle(st.Pid, t.Field(i).Name) {
					ts = append(ts, strings.Split(s, ":")[1])
				}
			}
			k := strings.ToLower(t.Field(i).Name)
			switch k {
			case "pid", "cid", "year":
				qw = qw.Where(fmt.Sprintf("%s = ?", k), value)
			case "area", "language":
				if strings.EqualFold(value.(string), "其它") {
					qw = qw.Where(fmt.Sprintf("%s NOT IN ?", k), ts)
					break
				}
				qw = qw.Where(fmt.Sprintf("%s = ?", k), value)
			case "plot":
				if strings.EqualFold(value.(string), "其它") {
					for _, t := range ts {
						qw = qw.Where("class_tag NOT LIKE ?", fmt.Sprintf("%%%v%%", t))
					}
					break
				}
				qw = qw.Where("class_tag LIKE ?", fmt.Sprintf("%%%v%%", value))
			case "sort":
				if strings.EqualFold(value.(string), "release_stamp") {
					qw = qw.Order(fmt.Sprintf("year DESC ,%v DESC", value))
					break
				}
				qw = qw.Order(fmt.Sprintf("%v DESC", value))
			default:
				break
			}
		}
	}

	// 返回分页参数
	GetPage(qw, page)
	// 查询具体的searchInfo 分页数据
	var sl []SearchInfo
	if err := qw.Limit(page.PageSize).Offset((page.Current - 1) * page.PageSize).Find(&sl).Error; err != nil {
		log.Println(err)
		return nil
	}
	return sl

}

// GetMovieListBySort 通过排序类型返回对应的影片基本信息
func GetMovieListBySort(t int, pid int64, page *Page) []MovieBasicInfo {
	var sl []SearchInfo
	qw := db.Mdb.Model(&SearchInfo{}).Where("pid", pid)
	switch t {
	case 0:
		qw = qw.Order("year DESC, release_stamp DESC")
	case 1:
		qw = qw.Order("year DESC, hits DESC")
	case 2:
		qw = qw.Order("year DESC, update_stamp DESC")
	}
	if err := qw.Limit(page.PageSize).Offset((page.Current - 1) * page.PageSize).Find(&sl).Error; err != nil {
		log.Println(err)
		return nil
	}
	return GetBasicInfoBySearchInfos(sl...)
}

// ================================= Manage 管理后台 =================================

func GetSearchPage(s SearchVo) []SearchInfo {
	// 构建 query查询条件
	query := db.Mdb.Model(&SearchInfo{})
	// 如果参数不为空则追加对应查询条件
	if s.Name != "" {
		query = query.Where("name LIKE ?", fmt.Sprintf("%%%s%%", s.Name))
	}
	// 分类ID为负数则默认不追加该条件
	if s.Cid > 0 {
		query = query.Where("cid = ?", s.Cid)
	} else if s.Pid > 0 {
		query = query.Where("pid = ?", s.Pid)
	}
	if s.Plot != "" {
		query = query.Where("class_tag LIKE ?", fmt.Sprintf("%%%s%%", s.Plot))
	}
	if s.Area != "" {
		query = query.Where("area = ?", s.Area)
	}
	if s.Language != "" {
		query = query.Where("language = ?", s.Language)
	}
	if int(s.Year) > time.Now().Year()-12 {
		query = query.Where("year = ?", s.Year)
	}
	switch s.Remarks {
	case "完结":
		query = query.Where("remarks IN ?", []string{"完结", "HD"})
	case "":
	default:
		query = query.Not(map[string]interface{}{"remarks": []string{"完结", "HD"}})
	}
	if s.BeginTime > 0 {
		query = query.Where("update_stamp >= ? ", s.BeginTime)
	}
	if s.EndTime > 0 {
		query = query.Where("update_stamp <= ? ", s.EndTime)
	}

	// 返回分页参数
	GetPage(query, s.Paging)
	// 查询具体的数据
	var sl []SearchInfo
	if err := query.Limit(s.Paging.PageSize).Offset((s.Paging.Current - 1) * s.Paging.PageSize).Find(&sl).Error; err != nil {
		log.Println(err)
		return nil
	}
	return sl

}

// GetSearchOptions 获取全部影片的检索标签信息
func GetSearchOptions(pid int64) map[string]interface{} {
	// 整合searchTag相关内容
	titles := db.Rdb.HGetAll(db.Cxt, fmt.Sprintf(config.SearchTitle, pid)).Val()
	// 处理单一分类的数据格式
	tagMap := make(map[string]interface{})
	for t, _ := range titles {
		switch t {
		// 只获取对应几个类型的标签
		case "Plot", "Area", "Language", "Year":
			tagMap[t] = HandleTagStr(t, GetTagsByTitle(pid, t)...)
		default:
		}
	}
	return tagMap
}

// ================================= 接口数据缓存 =================================

// DataCache  API请求 数据缓存
func DataCache(key string, data map[string]interface{}) {
	val, _ := json.Marshal(data)
	db.Rdb.Set(db.Cxt, key, val, time.Minute*30)
}

// GetCacheData 获取API接口的缓存数据
func GetCacheData(key string) map[string]interface{} {
	data := make(map[string]interface{})
	val, err := db.Rdb.Get(db.Cxt, key).Result()
	if err != nil || len(val) <= 0 {
		return nil
	}
	_ = json.Unmarshal([]byte(val), &data)
	return data
}

// RemoveCache 删除数据缓存
func RemoveCache(key string) {
	db.Rdb.Del(db.Cxt, key)
}

// ================================= OpenApi请求处理 =================================

func FindFilmIds(params map[string]string, page *Page) ([]int64, error) {
	var ids []int64
	query := db.Mdb.Model(&SearchInfo{}).Select("mid")
	for k, v := range params {
		// 如果 v 为空则直接 continue
		if len(v) <= 0 {
			continue
		}
		switch k {
		case "t":
			if cid, err := strconv.ParseInt(v, 10, 64); err == nil {
				query = query.Where("cid = ?", cid)
			}
		case "wd":
			query = query.Where("name like ?", fmt.Sprintf("%%%s%%", v))
		case "h":
			if h, err := strconv.ParseInt(v, 10, 64); err == nil {
				query = query.Where("update_stamp >= ?", time.Now().Unix()-h*3600)
			}
		}
	}
	// 返回分页参数
	var count int64
	query.Count(&count)
	page.Total = int(count)
	page.PageCount = int(page.Total+page.PageSize-1) / page.PageSize
	// 返回满足条件的ids
	err := query.Limit(page.PageSize).Offset((page.Current - 1) * page.PageSize).Order("update_stamp DESC").Find(&ids).Error
	return ids, err
}
