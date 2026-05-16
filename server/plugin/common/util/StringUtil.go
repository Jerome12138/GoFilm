package util

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// GenerateUUID 生成UUID
func GenerateUUID() (uuid string) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	uuid = fmt.Sprintf("%X-%X-%X-%X-%X",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return
}

// RandomString 生成指定长度两倍的随机字符串
func RandomString(length int) (uuid string) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	uuid = fmt.Sprintf("%x", b)
	return
}

// GenerateSalt 生成 length为16的随机字符串
func GenerateSalt() (uuid string) {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	uuid = fmt.Sprintf("%X", b)
	return
}

// PasswordEncrypt 历史密码加密算法 (password+salt) md5×3.
// 仅用于校验老用户旧哈希; 新用户/改密一律用 BcryptHash.
//
// 弱口令攻击下 md5×3 几秒内可彩虹表破解, 是 OWASP 反模式.
// 老库存量逐步迁移: 登录成功后透明 rehash 到 bcrypt (见 logic.UserLogin).
func PasswordEncrypt(password, salt string) string {
	b := []byte(fmt.Sprint(password, salt))
	var r [16]byte
	for i := 0; i < 3; i++ {
		r = md5.Sum(b)
		b = []byte(hex.EncodeToString(r[:]))
	}
	return hex.EncodeToString(r[:])
}

// BcryptCost: bcrypt 工作因子. 10 是 OWASP 当前推荐; 高于 12 在弱机器登录可见延迟.
const BcryptCost = 10

// IsBcryptHash 判断字符串是否是 bcrypt 哈希 ($2a$ / $2b$ / $2y$ 开头).
// 老 md5×3 哈希为 32 位 hex, 永远不会以 $ 开头, 可作为区分依据.
func IsBcryptHash(s string) bool {
	return strings.HasPrefix(s, "$2a$") || strings.HasPrefix(s, "$2b$") || strings.HasPrefix(s, "$2y$")
}

// BcryptHash 生成 bcrypt 哈希. salt 由 bcrypt 自动管理, 不再需要外部 salt.
func BcryptHash(password string) (string, error) {
	if password == "" {
		return "", errors.New("password is empty")
	}
	// bcrypt 上限 72 字节, 超过的会被静默截断; 这里显式拒绝避免迷惑.
	if len(password) > 72 {
		return "", errors.New("密码长度不能超过 72 字节")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// BcryptVerify 用 bcrypt 校验密码; 匹配返 true.
// 不匹配 / 哈希格式异常一律返 false, 不区分原因避免泄漏.
func BcryptVerify(password, hash string) bool {
	if hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// ParsePriKeyBytes 解析私钥
func ParsePriKeyBytes(buf []byte) (*rsa.PrivateKey, error) {
	p := &pem.Block{}
	p, buf = pem.Decode(buf)
	if p == nil {
		return nil, errors.New("private key parse  error")
	}
	return x509.ParsePKCS1PrivateKey(p.Bytes)
}

// ParsePubKeyBytes 解析公钥
func ParsePubKeyBytes(buf []byte) (*rsa.PublicKey, error) {
	p, _ := pem.Decode(buf)
	if p == nil {
		return nil, errors.New("parse publicKey content nil")
	}
	pubKey, err := x509.ParsePKCS1PublicKey(p.Bytes)
	if err != nil {
		return nil, errors.New("x509.ParsePKCS1PublicKey error")
	}
	return pubKey, nil
}

// ValidDomain 域名校验(http://example.xxx)
func ValidDomain(s string) bool {
	return regexp.MustCompile(`^(http|https)://[a-zA-Z0-9]+(\.[a-zA-Z0-9]+)*\.[a-z]{2,6}(:[0-9]{1,5})?$`).MatchString(s)
}

// ValidIPHost 校验是否符合http|https//ip 格式
func ValidIPHost(s string) bool {
	return regexp.MustCompile(`^(http|https)://(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})(:[0-9]{1,5})?$`).MatchString(s)
}

// ValidURL 校验http链接是否是符合规范的URL
func ValidURL(s string) bool {
	_, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}
	return true
}

func ValidPwd(s string) error {
	if len(s) < 8 || len(s) > 64 {
		return fmt.Errorf("密码长度不符合规范, 必须为 8-64 位")
	}
	// 分别校验数字 大小写字母和特殊字符
	num := `[0-9]{1}`
	l := `[a-z]{1}`
	u := `[A-Z]{1}`
	symbol := `[!@#~$%^&*()+|_]{1}`
	if b, err := regexp.MatchString(num, s); !b || err != nil {
		return errors.New("密码必须包含数字 ")
	}
	if b, err := regexp.MatchString(l, s); !b || err != nil {
		return errors.New("密码必须包含小写字母")
	}
	if b, err := regexp.MatchString(u, s); !b || err != nil {
		return errors.New("密码必须包含大写字母")
	}
	if b, err := regexp.MatchString(symbol, s); !b || err != nil {
		return errors.New("密码必须包含特殊字")
	}
	return nil
}
