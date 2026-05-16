package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

/*
 定义一些数据库存放的key值, 以及程序运行时的相关参数配置.
 凭据 / 网络地址类配置全部支持环境变量覆盖, 默认值仅供本地开发使用.
*/

// -------------------------System Config-----------------------------------
const (

	// ListenerPort web服务监听的端口
	ListenerPort = "3601"

	// MAXGoroutine max goroutine, 执行spider中对协程的数量限制
	MAXGoroutine = 10

	FilmPictureUploadDir = "./static/upload/gallery"
	FilmPictureUrlPath   = "/upload/pic/poster/"
	FilmPictureAccess    = "/api/upload/pic/poster/"
)

// -------------------------redis key-----------------------------------
const (
	CategoryTreeKey     = "CategoryTree"
	CategoryTreeExpired = time.Hour * 24 * 90
	MovieListInfoKey    = "MovieList:Cid%d"

	MovieDetailKey    = "MovieDetail:Cid%d:Id%d"
	MovieBasicInfoKey = "MovieBasicInfo:Cid%d:Id%d"

	MultipleSiteDetail = "MultipleSource:%s"

	SearchInfoTemp = "Search:SearchInfoTemp"

	SearchTitle = "Search:Pid%d:Title"
	SearchTag   = "Search:Pid%d:%s"

	VirtualPictureKey = "VirtualPicture"
	MaxScanCount      = 300
)

const (
	AuthUserClaims = "UserClaims"
)

// -------------------------manage 管理后台相关key----------------------------------
const (
	FilmSourceListKey   = "Config:Collect:FilmSource"
	ManageConfigExpired = time.Hour * 24 * 365 * 10
	SiteConfigBasic     = "SystemConfig:SiteConfig:Basic"

	FilmCrontabKey    = "Cron:Task:Film"
	DefaultUpdateSpec = "0 */20 * * * ?"
	DefaultUpdateTime = 3
)

// -------------------------Web API相关redis key-----------------------------------
const (
	IndexCacheKey = "IndexCache"
)

// -------------------------Database Connection Params-----------------------------------
const (
	SearchTableName       = "search"
	UserTableName         = "users"
	UserIdInitialVal      = 10000
	FileTableName         = "files"
	UserHistoryTableName  = "user_histories"
	UserFavoriteTableName = "user_favorites"
)

// 默认开发值. 生产环境必须用 env 覆盖, 否则启动期会大声告警.
const (
	defaultMysqlDsn      = "root:root@(127.0.0.1:3306)/FilmSite?charset=utf8mb4&parseTime=True&loc=Local"
	defaultRedisAddr     = "127.0.0.1:6379"
	defaultRedisPassword = ""
	defaultRedisDB       = 0
)

// 运行期实际生效的连接参数, 由 init() 从环境变量加载.
var (
	MysqlDsn      string
	RedisAddr     string
	RedisPassword string
	RedisDBNo     int
)

func init() {
	MysqlDsn = envOrDefault("MYSQL_DSN", defaultMysqlDsn, true)
	RedisAddr = envOrDefault("REDIS_ADDR", defaultRedisAddr, true)
	RedisPassword = envOrDefault("REDIS_PASSWORD", defaultRedisPassword, false)
	RedisDBNo = envInt("REDIS_DB", defaultRedisDB)
}

// envOrDefault 读 env; warnOnFallback=true 时如果回退到默认值则 log warn (用于关键凭据/网络配置).
func envOrDefault(key, def string, warnOnFallback bool) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	if warnOnFallback {
		log.Printf("WARN config: env %s 未设置, 回退到 DEV 默认值. 生产环境必须显式设置.", key)
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
		log.Printf("WARN config: env %s=%q 解析失败, 回退默认 %d", key, v, def)
	}
	return def
}
