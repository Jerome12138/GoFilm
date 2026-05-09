package param

import "testing"

func TestIsEmpty_Numeric(t *testing.T) {
	cases := []struct {
		v    any
		want bool
	}{
		{int(0), true}, {int(1), false},
		{int64(0), true}, {int64(-1), false},
		{float32(0), true}, {float64(1.5), false},
		{uint(0), true}, {uint8(2), false},
	}
	for _, c := range cases {
		if got := IsEmpty(c.v); got != c.want {
			t.Errorf("IsEmpty(%v) = %v, want %v", c.v, got, c.want)
		}
	}
}

func TestIsEmpty_String(t *testing.T) {
	if !IsEmpty("") {
		t.Errorf("empty string should be empty")
	}
	if IsEmpty("x") {
		t.Errorf("non-empty string should not be empty")
	}
}

func TestIsEmpty_Bool_FixedSemantics(t *testing.T) {
	// 修复后: false 视为空, true 视为非空
	if !IsEmpty(false) {
		t.Errorf("IsEmpty(false) should be true after semantic fix")
	}
	if IsEmpty(true) {
		t.Errorf("IsEmpty(true) should be false after semantic fix")
	}
}

func TestIsEmpty_OtherType(t *testing.T) {
	// 不支持的类型默认 false (非空)
	type s struct{ A int }
	if IsEmpty(s{}) {
		t.Errorf("struct should fall back to non-empty")
	}
}

func TestIsEmptyRe(t *testing.T) {
	if !IsEmptyRe[int](0) || IsEmptyRe[int](1) {
		t.Errorf("int IsEmptyRe wrong")
	}
	if !IsEmptyRe[string]("") || IsEmptyRe[string]("a") {
		t.Errorf("string IsEmptyRe wrong")
	}
	if !IsEmptyRe[bool](false) || IsEmptyRe[bool](true) {
		t.Errorf("bool IsEmptyRe wrong")
	}
}
