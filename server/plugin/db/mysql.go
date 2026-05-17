package db

import (
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"server/config"
)

var Mdb *gorm.DB

const (
	mysqlRetryAttempts = 5
	mysqlRetryBase     = 1 * time.Second

	// MySQL database/sql 连接池上限. GORM 默认无限连接, 高并发采集 (32-128 协程 + Web 请求)
	// 会瞬间打开几百个连接, 触及 MySQL 默认 max_connections=151 直接报错 "too many connections".
	// 这里给一个合理上限, 让 MySQL 端永远不会被打爆; 仍嫌不够可经 env 调高.
	mysqlMaxOpenConns    = 100
	mysqlMaxIdleConns    = 20
	mysqlConnMaxLifetime = 30 * time.Minute
)

// InitMysql 初始化 gorm 客户端. 与 redis 同样按指数退避重试 5 次, 仍失败 log.Fatal.
// 避免 docker compose 启动顺序导致的"mysql 慢半秒就死循环"问题.
func InitMysql() error {
	var lastErr error
	for i := 0; i < mysqlRetryAttempts; i++ {
		db, err := gorm.Open(mysql.New(mysql.Config{
			DSN:                       config.MysqlDsn,
			DefaultStringSize:         255,
			DisableDatetimePrecision:  true,
			DontSupportRenameIndex:    true,
			DontSupportRenameColumn:   true,
			SkipInitializeWithVersion: false,
		}), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true,
			},
		})
		if err == nil {
			// gorm.Open 不主动建连, 再 Ping 一次确认通路
			if sqlDB, e := db.DB(); e == nil {
				if pingErr := sqlDB.Ping(); pingErr == nil {
					// 配置连接池, 避免高并发场景下连接数爆炸
					sqlDB.SetMaxOpenConns(mysqlMaxOpenConns)
					sqlDB.SetMaxIdleConns(mysqlMaxIdleConns)
					sqlDB.SetConnMaxLifetime(mysqlConnMaxLifetime)
					Mdb = db
					return nil
				} else {
					lastErr = pingErr
				}
			} else {
				lastErr = e
			}
		} else {
			lastErr = err
		}
		wait := mysqlRetryBase << i
		log.Printf("WARN mysql: 连接失败 (%d/%d): %v, %s 后重试", i+1, mysqlRetryAttempts, lastErr, wait)
		time.Sleep(wait)
	}
	log.Fatalf("mysql: 连接失败, 最后错误: %v", lastErr)
	return lastErr // unreachable
}
