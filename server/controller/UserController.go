package controller

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"server/config"
	"server/logic"
	"server/model/system"
	"server/plugin/common/util"
)

/*
loginAllow: 登录端点的 per-IP token bucket 频控.

  - capacity = 10, refill = 10/60 token/s (稳态 10 次 / 60s, 突发 10)
  - 与 m3u8 代理频控共用同款数据结构 (本文件下面定义), 不引入新依赖
  - 多副本部署需在反代/网关层做更严的频控, 这里只兜底单进程
*/

const (
	loginRateCapacity = 10.0
	loginRateRefill   = 10.0 / 60.0
)

var (
	loginBuckets   = map[string]*tokenBucket{}
	loginBucketsMu sync.Mutex
)

func loginAllow(ip string) bool {
	loginBucketsMu.Lock()
	b := loginBuckets[ip]
	if b == nil {
		b = &tokenBucket{
			tokens:    loginRateCapacity,
			lastFill:  time.Now(),
			capacity:  loginRateCapacity,
			refillSec: loginRateRefill,
		}
		loginBuckets[ip] = b
	}
	loginBucketsMu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(b.lastFill).Seconds()
	b.tokens += elapsed * b.refillSec
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.lastFill = now
	if b.tokens < 1 {
		return false
	}
	b.tokens -= 1
	return true
}

// Login 用户登录接口 (普通用户和管理员共用同一鉴权流程, 角色由前端按 role 字段渲染入口).
// 响应体直接返回 LoginResult{userName, token, expires, role}, 前端从 body 取 token 写本地存储,
// 不再依赖 new-token 响应头.
//
// 频控: per-IP token bucket, 默认 10 次 / 60s; 超额直接 429, 抵御暴力撞库.
func Login(c *gin.Context) {
	if !loginAllow(c.ClientIP()) {
		system.CustomResult(http.StatusTooManyRequests, system.FAILED, nil, "登录请求过于频繁, 请稍候再试", c)
		return
	}
	var u system.User
	if err := c.ShouldBindJSON(&u); err != nil {
		system.Failed("登录信息异常!!!", c)
		return
	}
	if len(u.UserName) <= 0 || len(u.Password) <= 0 {
		system.Failed("用户名和密码信息不能为空", c)
		return
	}
	res, err := logic.UL.UserLogin(u.UserName, u.Password)
	if err != nil {
		log.Printf("login fail ip=%s user=%q: %v", c.ClientIP(), u.UserName, err)
		system.Failed(err.Error(), c) // err 都是受控文案 (用户名不存在 / 密码错), 不泄漏底层
		return
	}
	system.Success(res, "登录成功", c)
}

// ManageUserCreate 管理员后台创建用户账号.
// 路由层用 RequireAdmin 限制只有管理员可以调用; body.role 决定新账号是普通用户还是管理员.
// 不再向公网暴露公共注册.
func ManageUserCreate(c *gin.Context) {
	var p logic.RegisterParams
	if err := c.ShouldBindJSON(&p); err != nil {
		system.Failed("请求参数异常", c)
		return
	}
	info, err := logic.UL.CreateAccount(p)
	if err != nil {
		system.Failed(err.Error(), c)
		return
	}
	system.Success(info, "用户创建成功", c)
}

// ManageUserList 管理员查看用户列表
func ManageUserList(c *gin.Context) {
	current, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
	if current < 1 {
		current = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	page := &system.Page{Current: current, PageSize: pageSize}
	list := logic.UL.ListUsers(page)
	system.Success(gin.H{"list": list, "page": page}, "用户列表获取成功", c)
}

// Logout 退出登录
func Logout(c *gin.Context) {
	// 获取已登录的用户信息
	v, ok := c.Get(config.AuthUserClaims)
	if !ok {
		system.Failed("请求失败,登录信息获取异常!!!", c)
		return
	}
	// 清除redis中存储的对应token
	uc, ok := v.(*system.UserClaims)
	if !ok {
		system.Failed("注销失败, 身份信息格式化异常!!!", c)
		return
	}
	err := system.ClearUserToken(uc.UserID)
	if err != nil {
		log.Println("user logOut err: ", err)
	}
	system.SuccessOnlyMsg("已退出登录!!!", c)
}

// UserPasswordChange 修改用户密码
func UserPasswordChange(c *gin.Context) {
	// 接收密码修改参数
	var params map[string]string
	if err := c.ShouldBindJSON(&params); err != nil {
		system.Failed("参数校验失败!!!", c)
		return
	}
	// 校验参数是否存在空值
	if params["password"] == "" || params["newPassword"] == "" {
		system.Failed("密码不能为空!!!", c)
		return
	}
	// 校验新密码是否符合规范
	if err := util.ValidPwd(params["newPassword"]); err != nil {
		system.Failed(fmt.Sprint("密码格式校验失败: ", err.Error()), c)
		return
	}
	// 获取已登录的用户信息
	v, ok := c.Get(config.AuthUserClaims)
	if !ok {
		system.Failed("操作失败,登录信息异常!!!", c)
		return
	}
	// 从context中获取用户的登录信息
	uc, ok := v.(*system.UserClaims)
	if !ok {
		system.Failed("操作失败, 用户身份信息异常", c)
		return
	}
	if err := logic.UL.ChangePassword(uc.UserName, params["password"], params["newPassword"]); err != nil {
		system.Failed(fmt.Sprint("密码修改失败: ", err.Error()), c)
		return
	}
	// 密码修改成功后不主动使token失效, 以免影响体验
	system.SuccessOnlyMsg("密码修改成功", c)
}

func UserInfo(c *gin.Context) {
	// 从context中获取用户的相关信息
	v, ok := c.Get(config.AuthUserClaims)
	if !ok {
		system.Failed("用户信息获取失败, 未获取到用户授权信息", c)
		return
	}
	uc, ok := v.(*system.UserClaims)
	if !ok {
		system.Failed("用户信息获取失败, 户授权信息异常", c)
		return
	}
	// 通过用户ID获取用户基本信息
	info := logic.UL.GetUserInfo(uc.UserID)
	system.Success(info, "成功获取用户信息", c)
}
