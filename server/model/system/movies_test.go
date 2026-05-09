package system

import (
	"strings"
	"testing"
)

func TestGenerateHashKey_StringNormalization(t *testing.T) {
	cases := []struct {
		name string
		a    string
		b    string
		want bool // a 和 b 应该 hash 相等
	}{
		{"忽略空格", "天龙 八部", "天龙八部", true},
		{"去除别名后缀 ～xxx～", "天龙八部～国语版～", "天龙八部", true},
		{"去除 ASCII 首尾标点", ".天龙八部.", "天龙八部", true}, // POSIX [:punct:] 仅 ASCII
		{"季后缀压缩", "天龙八部第一季高清版", "天龙八部第一季", true}, // "季.*" 折叠到 "季"
		{"完全不同的名称", "天龙八部", "笑傲江湖", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h1 := GenerateHashKey(c.a)
			h2 := GenerateHashKey(c.b)
			if (h1 == h2) != c.want {
				t.Errorf("GenerateHashKey(%q)=%s vs GenerateHashKey(%q)=%s want equal=%v", c.a, h1, c.b, h2, c.want)
			}
		})
	}
}

func TestGenerateHashKey_AcceptsIntAndInt64(t *testing.T) {
	hStr := GenerateHashKey("12345")
	hInt := GenerateHashKey(12345)
	hInt64 := GenerateHashKey(int64(12345))
	if hStr != hInt || hStr != hInt64 {
		t.Errorf("hash should be identical across string/int/int64 of same value: %s/%s/%s", hStr, hInt, hInt64)
	}
}

func TestConvertSearchInfo_BasicFields(t *testing.T) {
	d := MovieDetail{
		Id:  100,
		Cid: 6,
		Pid: 1,
		Name: "天龙八部",
		MovieDescriptor: MovieDescriptor{
			SubTitle:    "Demi-Gods",
			CName:       "电视剧",
			ClassTag:    "武侠,古装",
			Area:        "中国大陆",
			Language:    "国语",
			ReleaseDate: "2003-09-15",
			DbScore:     "8.5",
			Hits:        1234,
			UpdateTime:  "2024-01-01 12:00:00",
			AddTime:     1700000000,
			State:       "正片",
			Remarks:     "完结",
		},
	}
	s := ConvertSearchInfo(d)
	if s.Mid != 100 || s.Cid != 6 || s.Pid != 1 {
		t.Errorf("id mapping wrong: %+v", s)
	}
	if s.Score != 8.5 {
		t.Errorf("score parse failed, got %v", s.Score)
	}
	if s.Year != 2003 {
		t.Errorf("year parse failed, got %v", s.Year)
	}
	if s.Hits != 1234 {
		t.Errorf("hits mapping wrong: %v", s.Hits)
	}
	if s.ReleaseStamp != 1700000000 {
		t.Errorf("ReleaseStamp should fall back to AddTime when ReleaseDate has no full timestamp, got %v", s.ReleaseStamp)
	}
}

func TestConvertSearchInfo_BadReleaseDateGivesYearZero(t *testing.T) {
	d := MovieDetail{
		MovieDescriptor: MovieDescriptor{ReleaseDate: "未知"},
	}
	s := ConvertSearchInfo(d)
	if s.Year != 0 {
		t.Errorf("expected year=0 for unparsable release date, got %v", s.Year)
	}
}

func TestBuildMovieBasicInfo_ProjectsExpectedFields(t *testing.T) {
	d := MovieDetail{
		Id: 9, Cid: 7, Pid: 2,
		Name:    "测试",
		Picture: "http://x/a.png",
		MovieDescriptor: MovieDescriptor{
			SubTitle: "sub",
			CName:    "电影",
			Actor:    "甲,乙",
			Director: "导演",
			Blurb:    "简介",
			Remarks:  "HD",
			Area:     "中国",
			Year:     "2024",
			State:    "正片",
		},
	}
	b := buildMovieBasicInfo(d)
	if b.Id != 9 || b.Cid != 7 || b.Pid != 2 {
		t.Errorf("ids: %+v", b)
	}
	if !strings.EqualFold(b.Year, "2024") {
		t.Errorf("year: %v", b.Year)
	}
	if b.Picture != d.Picture {
		t.Errorf("picture not projected: %v", b.Picture)
	}
}
