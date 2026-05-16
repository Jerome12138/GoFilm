package util

import (
	"strings"
	"testing"
)

func TestPasswordEncrypt_StableAndDifferentSalts(t *testing.T) {
	enc1 := PasswordEncrypt("Pass#1234", "salt-A")
	enc2 := PasswordEncrypt("Pass#1234", "salt-A")
	if enc1 != enc2 {
		t.Fatalf("expected deterministic output for same (password,salt), got %s vs %s", enc1, enc2)
	}
	if len(enc1) != 32 {
		t.Fatalf("expected 32-char md5 hex, got len=%d (%s)", len(enc1), enc1)
	}
	enc3 := PasswordEncrypt("Pass#1234", "salt-B")
	if enc1 == enc3 {
		t.Fatalf("different salt must produce different hash")
	}
}

func TestGenerateSalt_LenAndUnique(t *testing.T) {
	s1 := GenerateSalt()
	s2 := GenerateSalt()
	// 8 random bytes -> hex(uppercase) of 16 chars
	if len(s1) != 16 || len(s2) != 16 {
		t.Fatalf("salt length should be 16, got %d / %d", len(s1), len(s2))
	}
	if s1 == s2 {
		t.Fatalf("two consecutive salts collided: %s", s1)
	}
	if strings.ToUpper(s1) != s1 {
		t.Fatalf("salt should be uppercase hex, got %s", s1)
	}
}

func TestRandomString_LenIsTwiceInput(t *testing.T) {
	r := RandomString(8)
	if len(r) != 16 {
		t.Fatalf("RandomString(8) should produce 16 hex chars, got len=%d", len(r))
	}
}

func TestGenerateUUID_Pattern(t *testing.T) {
	u := GenerateUUID()
	// 期望形如 XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX (大写 hex)
	if got, want := strings.Count(u, "-"), 4; got != want {
		t.Fatalf("UUID should have 4 dashes, got %d in %q", got, u)
	}
}

func TestValidDomain(t *testing.T) {
	cases := map[string]bool{
		"http://example.com":         true,
		"https://api.example.com":    true,
		"https://a.b.c.example.com":  true,
		"http://example.com:8080":    true,
		"ftp://example.com":          false,
		"example.com":                false,
		"http://192.168.1.1":         false, // ValidDomain 不允许纯 IP
		"http://example":             false, // 顶级域不能少于 2 字符
	}
	for in, want := range cases {
		if got := ValidDomain(in); got != want {
			t.Errorf("ValidDomain(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestValidIPHost(t *testing.T) {
	cases := map[string]bool{
		"http://192.168.1.1":      true,
		"https://10.0.0.1:3000":   true,
		"http://example.com":      false,
		"http://1.2.3":            false,
		"http://1.2.3.4.5":        false,
	}
	for in, want := range cases {
		if got := ValidIPHost(in); got != want {
			t.Errorf("ValidIPHost(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestValidURL(t *testing.T) {
	if !ValidURL("https://example.com/path?x=1") {
		t.Errorf("expected valid URL")
	}
	if ValidURL("::not-a-url") {
		t.Errorf("expected invalid URL to fail")
	}
}

func TestValidPwd(t *testing.T) {
	cases := []struct {
		pwd     string
		wantErr bool
		reason  string
	}{
		{"Aa1@aaaa", false, "正常 8 位含数字大小写特殊字符"},
		{"Aa1@a", true, "过短"},
		{"Aaaaaaaa", true, "缺数字与特殊字符"},
		{"AA1@AAAA", true, "缺小写"},
		{"aa1@aaaa", true, "缺大写"},
		{"Aa1aaaaa", true, "缺特殊字符"},
		{"Aa1@aaaaaaaaa", false, "13 位仍合法 (上限放宽到 64)"},
		{"Aa1@" + strings.Repeat("a", 60), false, "64 字符合法 (含 4 字符前缀)"},
		{"Aa1@" + strings.Repeat("a", 61), true, "65 字符超 64 上限"},
	}
	for _, c := range cases {
		err := ValidPwd(c.pwd)
		if (err != nil) != c.wantErr {
			t.Errorf("ValidPwd(%q): got err=%v, want err=%v (%s)", c.pwd, err, c.wantErr, c.reason)
		}
	}
}

func TestParsePubAndPrivKey_RejectInvalid(t *testing.T) {
	if _, err := ParsePriKeyBytes([]byte("not pem")); err == nil {
		t.Errorf("expected error for invalid private key")
	}
	if _, err := ParsePubKeyBytes([]byte("not pem")); err == nil {
		t.Errorf("expected error for invalid public key")
	}
}
