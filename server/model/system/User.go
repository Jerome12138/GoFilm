package system

import (
	"fmt"
	"gorm.io/gorm"
	"log"
	"server/config"
	"server/plugin/common/util"
	"server/plugin/db"
)

// 用户角色常量
const (
	RoleNormal = 0 // 普通用户
	RoleAdmin  = 1 // 管理员
)

type User struct {
	gorm.Model
	UserName string `json:"userName"` // 用户名
	Password string `json:"password"` // 密码
	Salt     string `json:"salt"`     // 盐值
	Email    string `json:"email"`    // 邮箱
	Gender   int    `json:"gender"`   // 性别
	NickName string `json:"nickName"` // 昵称
	Avatar   string `json:"avatar"`   // 头像
	Status   int    `json:"status"`   // 状态
	Role     int    `json:"role"`     // 角色: 0 普通用户, 1 管理员
	Reserve1 string `json:"reserve1"` // 预留字段 3
	Reserve2 string `json:"reserve2"` // 预留字段 2
	Reserve3 string `json:"reserve3"` // 预留字段 1
	//LastLongTime time.Time `json:"LastLongTime"` // 最后登录时间
}

// TableName 设置user表的表名
func (u *User) TableName() string {
	return config.UserTableName
}

// CreateUserTable 创建/迁移用户表.
// 已存在时仍然调用 AutoMigrate, 用于在不重建表的前提下追加新字段 (例如 role).
func CreateUserTable() {
	if db.Mdb == nil {
		return
	}
	u := &User{}
	created := !ExistUserTable()
	if err := db.Mdb.AutoMigrate(u); err != nil {
		log.Println("AutoMigrate User Failed: ", err)
		return
	}
	if created {
		db.Mdb.Exec(fmt.Sprintf("alter table %s auto_Increment=%d", u.TableName(), config.UserIdInitialVal))
	}
}

// ExistUserTable 判断表中是否存在User表
func ExistUserTable() bool {
	return db.Mdb.Migrator().HasTable(&User{})
}

// InitAdminAccount 初始化 admin 账号.
// 不存在 → 直接以 RoleAdmin 创建;
// 已存在但 Role == 0 → 升级为管理员 (兼容老库, 避免引入 role 字段后管理员丢权限).
func InitAdminAccount() {
	user := GetUserByNameOrEmail("admin")
	if user != nil {
		if user.Role != RoleAdmin && db.Mdb != nil {
			db.Mdb.Model(&User{}).Where("id = ?", user.ID).Update("role", RoleAdmin)
		}
		return
	}
	u := &User{
		UserName: "admin",
		Password: "admin",
		Salt:     util.GenerateSalt(),
		Email:    "administrator@gmail.com",
		Gender:   2,
		NickName: "Zero",
		Avatar:   "empty",
		Status:   0,
		Role:     RoleAdmin,
	}
	u.Password = util.PasswordEncrypt(u.Password, u.Salt)
	db.Mdb.Create(u)
}

// GetUserByNameOrEmail 查询 username || email 对应的账户信息
func GetUserByNameOrEmail(userName string) *User {
	var u *User
	if err := db.Mdb.Where("user_name = ? OR email = ?", userName, userName).First(&u).Error; err != nil {
		log.Println(err)
		return nil
	}
	return u
}

func GetUserById(id uint) User {
	var user = User{Model: gorm.Model{ID: id}}
	db.Mdb.First(&user)
	return user
}

// UpdateUserInfo 更新用户信息
func UpdateUserInfo(u User) {
	// 值更新允许修改的部分字段, 零值会在更新时被自动忽略
	db.Mdb.Model(&u).Updates(User{Password: u.Password, Email: u.Email, NickName: u.NickName, Status: u.Status})
}

// ExistUserByName 用户名是否被占用
func ExistUserByName(userName string) bool {
	if db.Mdb == nil {
		return false
	}
	var count int64
	db.Mdb.Model(&User{}).Where("user_name = ?", userName).Count(&count)
	return count > 0
}

// ExistUserByEmail 邮箱是否被占用 (空字符串视为未占用)
func ExistUserByEmail(email string) bool {
	if db.Mdb == nil || email == "" {
		return false
	}
	var count int64
	db.Mdb.Model(&User{}).Where("email = ?", email).Count(&count)
	return count > 0
}

// CreateUser 写入新用户记录
func CreateUser(u *User) error {
	if db.Mdb == nil {
		return ErrDBNotInitialized
	}
	return db.Mdb.Create(u).Error
}

// ListUsers 管理员查看用户分页 (按 ID 升序, 不返回 password / salt)
func ListUsers(page *Page) []UserInfoVo {
	if db.Mdb == nil {
		return nil
	}
	q := db.Mdb.Model(&User{})
	GetPage(q, page)
	var users []User
	if err := q.Limit(page.PageSize).Offset((page.Current-1)*page.PageSize).
		Order("id ASC").Find(&users).Error; err != nil {
		log.Println("ListUsers err: ", err)
		return nil
	}
	out := make([]UserInfoVo, 0, len(users))
	for _, u := range users {
		out = append(out, UserInfoVo{
			Id: u.ID, UserName: u.UserName, Email: u.Email,
			Gender: u.Gender, NickName: u.NickName, Avatar: u.Avatar,
			Status: u.Status, Role: u.Role,
		})
	}
	return out
}
