package logic

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"server/config"
	"server/model/system"
	"server/plugin/common/util"
)

type UserLogic struct {
}

var UL *UserLogic

// reUserName 用户名: 4-20 位英文/数字/下划线
var reUserName = regexp.MustCompile(`^[A-Za-z0-9_]{4,20}$`)

// reEmail 简单的邮箱格式校验, 满足"username@host.tld"形态即可
var reEmail = regexp.MustCompile(`^[\w\-.+]+@[\w\-]+(?:\.[\w\-]+)+$`)

// RegisterParams 用户注册请求参数
type RegisterParams struct {
	UserName string `json:"userName"`
	Password string `json:"password"`
	Email    string `json:"email"`
	NickName string `json:"nickName"`
	// Role 仅在管理员后台创建用户时生效, 公共注册路径不开放该字段.
	Role int `json:"role"`
}

// LoginResult 登录成功后下发的结构化数据.
// 前端凭 Token 注入 Authorization: Bearer <token>, Expires 用于本地刷新前判断,
// Role 给前端按权限渲染入口.
type LoginResult struct {
	UserName string `json:"userName"`
	Token    string `json:"token"`
	Expires  int64  `json:"expires"` // unix 秒, token 过期时间
	Role     int    `json:"role"`
}

// UserLogin 用户登录, 校验通过后生成 token 并落 redis, 返回完整 LoginResult.
func (ul *UserLogic) UserLogin(account, password string) (LoginResult, error) {
	u := system.GetUserByNameOrEmail(account)
	if u == nil {
		return LoginResult{}, errors.New("用户信息不存在")
	}
	if util.PasswordEncrypt(password, u.Salt) != u.Password {
		return LoginResult{}, errors.New("用户名或密码错误")
	}
	token, err := system.GenToken(u.ID, u.UserName, u.Role)
	if err != nil {
		return LoginResult{}, err
	}
	if err := system.SaveUserToken(token, u.ID); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{
		UserName: u.UserName,
		Token:    token,
		Expires:  time.Now().Add(config.AuthTokenExpires * time.Hour).Unix(),
		Role:     u.Role,
	}, nil
}

// CreateAccount 创建用户账号 (管理员后台调用).
// 校验顺序: 格式 -> 唯一性 -> 写库.
// 不返回 password / salt, 调用方只取脱敏后的展示信息.
// p.Role 决定新建账号的角色, 默认 0 (普通用户). 调用方应在路由层用 RequireAdmin
// 限制只有管理员能传 RoleAdmin.
func (ul *UserLogic) CreateAccount(p RegisterParams) (system.UserInfoVo, error) {
	p.UserName = strings.TrimSpace(p.UserName)
	p.Email = strings.TrimSpace(p.Email)
	p.NickName = strings.TrimSpace(p.NickName)

	if !reUserName.MatchString(p.UserName) {
		return system.UserInfoVo{}, errors.New("用户名格式不合法 (4-20 位英文/数字/下划线)")
	}
	if err := util.ValidPwd(p.Password); err != nil {
		return system.UserInfoVo{}, err
	}
	if p.Email != "" && !reEmail.MatchString(p.Email) {
		return system.UserInfoVo{}, errors.New("邮箱格式不正确")
	}
	if p.Role != system.RoleNormal && p.Role != system.RoleAdmin {
		return system.UserInfoVo{}, errors.New("角色取值非法 (0=普通用户, 1=管理员)")
	}
	if system.ExistUserByName(p.UserName) {
		return system.UserInfoVo{}, errors.New("用户名已被占用")
	}
	if system.ExistUserByEmail(p.Email) {
		return system.UserInfoVo{}, errors.New("邮箱已被注册")
	}

	salt := util.GenerateSalt()
	if p.NickName == "" {
		p.NickName = p.UserName
	}
	u := &system.User{
		UserName: p.UserName,
		Password: util.PasswordEncrypt(p.Password, salt),
		Salt:     salt,
		Email:    p.Email,
		NickName: p.NickName,
		Avatar:   "",
		Status:   0,
		Role:     p.Role,
	}
	if err := system.CreateUser(u); err != nil {
		return system.UserInfoVo{}, errors.New("创建失败, 请稍后重试")
	}
	return system.UserInfoVo{
		Id: u.ID, UserName: u.UserName, Email: u.Email,
		Gender: u.Gender, NickName: u.NickName, Avatar: u.Avatar, Status: u.Status, Role: u.Role,
	}, nil
}

// ListUsers 管理员查看用户分页列表
func (ul *UserLogic) ListUsers(page *system.Page) []system.UserInfoVo {
	return system.ListUsers(page)
}

// HistoryUpsertParams 上报观看进度的请求参数
type HistoryUpsertParams struct {
	Mid          int64  `json:"mid"`
	Cid          int64  `json:"cid"`
	Pid          int64  `json:"pid"`
	Name         string `json:"name"`
	Picture      string `json:"picture"`
	PlayFrom     string `json:"playFrom"`
	PlayFromName string `json:"playFromName"`
	Episode      int    `json:"episode"`
	EpisodeName  string `json:"episodeName"`
	Progress     int64  `json:"progress"`
	Duration     int64  `json:"duration"`
}

// UpsertHistory 上报或更新一条观看记录
func (ul *UserLogic) UpsertHistory(uid uint, p HistoryUpsertParams) error {
	if p.Mid <= 0 || strings.TrimSpace(p.Name) == "" {
		return errors.New("影片信息不完整")
	}
	return system.UpsertUserHistory(&system.UserHistory{
		UserID:       uid,
		Mid:          p.Mid,
		Cid:          p.Cid,
		Pid:          p.Pid,
		Name:         p.Name,
		Picture:      p.Picture,
		PlayFrom:     p.PlayFrom,
		PlayFromName: p.PlayFromName,
		Episode:      p.Episode,
		EpisodeName:  p.EpisodeName,
		Progress:     p.Progress,
		Duration:     p.Duration,
	})
}

// ListHistory 用户观看历史分页
func (ul *UserLogic) ListHistory(uid uint, page *system.Page) []system.UserHistory {
	return system.ListUserHistory(uid, page)
}

// DeleteHistory 删除某条历史 (优先按 id, mid 兜底)
func (ul *UserLogic) DeleteHistory(uid uint, id uint, mid int64) error {
	if id > 0 {
		return system.DeleteUserHistoryByID(uid, id)
	}
	if mid > 0 {
		return system.DeleteUserHistoryByMid(uid, mid)
	}
	return errors.New("缺少 id 或 mid")
}

// ClearHistory 清空当前用户全部历史
func (ul *UserLogic) ClearHistory(uid uint) error {
	return system.ClearUserHistory(uid)
}

// FavoriteParams 添加收藏的请求参数
type FavoriteParams struct {
	Mid     int64  `json:"mid"`
	Cid     int64  `json:"cid"`
	Pid     int64  `json:"pid"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Remarks string `json:"remarks"`
}

// AddFavorite 添加收藏 (幂等, 已存在则忽略)
func (ul *UserLogic) AddFavorite(uid uint, p FavoriteParams) error {
	if p.Mid <= 0 || strings.TrimSpace(p.Name) == "" {
		return errors.New("影片信息不完整")
	}
	return system.AddUserFavorite(&system.UserFavorite{
		UserID:  uid,
		Mid:     p.Mid,
		Cid:     p.Cid,
		Pid:     p.Pid,
		Name:    p.Name,
		Picture: p.Picture,
		Remarks: p.Remarks,
	})
}

// RemoveFavorite 取消收藏
func (ul *UserLogic) RemoveFavorite(uid uint, mid int64) error {
	if mid <= 0 {
		return errors.New("缺少 mid")
	}
	return system.RemoveUserFavorite(uid, mid)
}

// ListFavorite 收藏分页
func (ul *UserLogic) ListFavorite(uid uint, page *system.Page) []system.UserFavorite {
	return system.ListUserFavorite(uid, page)
}

// CheckFavorite 是否已收藏
func (ul *UserLogic) CheckFavorite(uid uint, mid int64) bool {
	return system.IsUserFavorite(uid, mid)
}

// UserLogout 用户退出登录 注销
func (ul *UserLogic) UserLogout() {
	// 通过用户ID清除Redis中的token信息

}

// ChangePassword 修改密码
func (ul *UserLogic) ChangePassword(account, password, newPassword string) error {
	// 根据 username 或 email 查询用户信息
	var u *system.User = system.GetUserByNameOrEmail(account)
	// 用户信息不存在则返回提示信息
	if u == nil {
		return errors.New(" 用户信息不存在!!!")
	}
	// 首先校验用户的旧密码是否正确
	if util.PasswordEncrypt(password, u.Salt) != u.Password {
		return errors.New("原密码校验失败")
	}
	// 密码校验正确则生成新的用户信息
	newUser := system.User{}
	newUser.ID = u.ID
	// 将新密码进行加密
	newUser.Password = util.PasswordEncrypt(newPassword, u.Salt)
	// 更新用户信息
	system.UpdateUserInfo(newUser)
	return nil
}

func (ul *UserLogic) GetUserInfo(id uint) system.UserInfoVo {
	// 通过用户ID查询对应的用户信息
	u := system.GetUserById(id)
	// 去除user信息中的不必要信息, 角色一并带回前端用于按权限渲染
	var vo = system.UserInfoVo{
		Id: u.ID, UserName: u.UserName, Email: u.Email,
		Gender: u.Gender, NickName: u.NickName, Avatar: u.Avatar,
		Status: u.Status, Role: u.Role,
	}
	return vo
}
