package main

import (
	"fmt"
	"server/config"
	"server/model/system"
	"server/plugin/SystemInit"
	"server/plugin/db"
	"server/router"
)

func init() {
	// 执行初始化前等待20s , 让mysql服务完成初始化指令
	//time.Sleep(time.Second * 20)
	//初始化redis客户端
	err := db.InitRedisConn()
	if err != nil {
		panic(err)
	}
	// 初始化mysql
	err = db.InitMysql()
	if err != nil {
		panic(err)
	}
}

func main() {
	start()
}

func start() {

	// 启动前先执行数据库内容的初始化工作
	DefaultDataInit()
	// 开启路由监听
	r := router.SetupRouter()
	_ = r.Run(fmt.Sprintf(":%s", config.ListenerPort))
}

func DefaultDataInit() {
	// 首次部署: users 表尚未建出 → 同时初始化网站基本配置 (避免覆盖运维改过的配置)
	firstBoot := !system.ExistUserTable()
	// TableInIt 内部按表分别 HasTable 守卫, 重复执行无害;
	// 用于在已上线系统增量增加 user_histories / user_favorites 等新表.
	SystemInit.TableInIt()
	if firstBoot {
		SystemInit.BasicConfigInit()
	}
	// 初始化影视来源列表信息
	SystemInit.SpiderInit()
}
