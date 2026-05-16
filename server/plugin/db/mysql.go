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
