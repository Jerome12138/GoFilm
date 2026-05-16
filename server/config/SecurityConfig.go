package config

import (
	"log"
	"os"
)

// JWT 密钥 + token 配置.
//
// 生产部署:
//   - 必须通过环境变量 JWT_PRIVATE_KEY / JWT_PUBLIC_KEY 注入 PEM 文本
//     (docker-compose / k8s secrets / .env 文件均可)
//   - 不再依赖代码中编译进二进制的默认密钥; 默认值仅供本地开发使用,
//     且每次启动会大声告警
//
// 旧代码把私钥直接 commit 到仓库 → 任何拿到代码的人都能伪造任意账户 token.
// 升级后请轮换密钥并重新登录所有用户.

var (
	// PrivateKey / PublicKey 由 init() 从环境变量读出, 找不到时回退到 devPrivateKey/devPublicKey.
	// 类型保持 string 以兼容 [] byte(config.PrivateKey) 这种历史调用.
	PrivateKey string
	PublicKey  string
)

// 仅用于本地开发的 demo 密钥. 任何被 commit 进仓库的密钥都视为已泄漏.
// 生产环境必须用 JWT_PRIVATE_KEY / JWT_PUBLIC_KEY 覆盖.
const devPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIBPAIBAAJBANNnshoUaT2gFNrihmFdmC1cBCs1XLFc5Fn3MfNOR3aOGDO0ohXl
bku6Ir/qITN/yeH5pY34WEcETet3YhESpE8CAwEAAQJBAI7Ekdfg/u26RTtJDd2F
WrcPVFVl1TKGfERxl08sB0D9HLvUSBfAEg/UpfWSQ57aSJ9b0gVKmDhgF8FymuUV
v2kCIQDzXXSZ/oeKmqObwad0Fa82IFof3LeZdpbrjyz3w45JDQIhAN5hdmuW+y2w
UgSy0o4zGFsEG/RBZsvVnSSfkdR47dPLAiEA2XbPNLQu5fnc7NeVDLQ7xsAOCJ6w
KR/BKGjeI9/JCxkCIQCjMkU0ec2FXxMhzZXFs2uZR6+4FdL5nZ9ABDaCBekK9wIg
XEfd11qabi9jPrbsOVNZCTk51B7Ug0ZwGyn0BA8Jlo0=
-----END RSA PRIVATE KEY-----
`

const devPublicKey = `-----BEGIN RSA PUBLIC KEY-----
MEgCQQDTZ7IaFGk9oBTa4oZhXZgtXAQrNVyxXORZ9zHzTkd2jhgztKIV5W5LuiK/
6iEzf8nh+aWN+FhHBE3rd2IREqRPAgMBAAE=
-----END RSA PUBLIC KEY-----
`

const (
	Issuer           = "GoFilm"
	AuthTokenExpires = 10 * 24 // 单位 h
	UserTokenKey     = "User:Token:%d"
)

func init() {
	PrivateKey = loadJWTKey("JWT_PRIVATE_KEY", "JWT_PRIVATE_KEY_FILE", devPrivateKey, "private")
	PublicKey = loadJWTKey("JWT_PUBLIC_KEY", "JWT_PUBLIC_KEY_FILE", devPublicKey, "public")
}

// loadJWTKey 按以下顺序解析 JWT 密钥:
//  1. envContent (例 JWT_PRIVATE_KEY) — PEM 文本直接注入
//  2. envFile (例 JWT_PRIVATE_KEY_FILE) — 指向 PEM 文件路径, 适合 docker/k8s secrets 挂载
//  3. 都没有 → 回退 dev 密钥, 启动期大声告警
//
// 返回值始终非空, 避免上层 ParsePriKeyBytes 处理空字符串场景.
func loadJWTKey(envContent, envFile, devFallback, label string) string {
	if v := os.Getenv(envContent); v != "" {
		return v
	}
	if path := os.Getenv(envFile); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("config: 读取 JWT %s 密钥文件失败 %s: %v", label, path, err)
		}
		return string(b)
	}
	log.Printf("WARN config: JWT %s 密钥未通过 env 注入, 回退到 DEV 密钥. 生产环境严禁如此 — 请设置 %s 或 %s.", label, envContent, envFile)
	return devFallback
}
