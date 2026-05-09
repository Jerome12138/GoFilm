package conver

import (
	"testing"

	"github.com/stretchr/testify/require"
	"server/model/collect"
	"server/model/system"
)

func TestGenCategoryTree_BuildsHierarchy(t *testing.T) {
	list := []collect.FilmClass{
		{TypeID: 1, TypePid: 0, TypeName: "电影"},
		{TypeID: 6, TypePid: 1, TypeName: "动作"},
		{TypeID: 2, TypePid: 0, TypeName: "电视剧"},
	}
	tree := GenCategoryTree(list)
	require.NotNil(t, tree)
	require.Equal(t, int64(0), tree.Id)
	require.Len(t, tree.Children, 2)

	movie := tree.Children[0]
	require.Equal(t, "电影", movie.Name)
	require.Len(t, movie.Children, 1)
	require.Equal(t, "动作", movie.Children[0].Name)
}

func TestGenCategoryTree_ChildBeforeParentDoesNotPanic(t *testing.T) {
	// 历史 bug: 子节点先到、父节点 nil 时会 panic. 现在应该用占位补齐.
	list := []collect.FilmClass{
		{TypeID: 99, TypePid: 50, TypeName: "未知子类"}, // 父节点 50 还没出现
		{TypeID: 50, TypePid: 0, TypeName: "影片"},     // 后到的父节点
	}
	tree := GenCategoryTree(list)
	require.NotNil(t, tree)

	// 父节点应该被补全 Name=影片, 包含子节点 99
	var parent *system.CategoryTree
	for _, c := range tree.Children {
		if c.Id == 50 {
			parent = c
			break
		}
	}
	require.NotNil(t, parent, "parent 50 should attach under root")
	require.Equal(t, "影片", parent.Name)
	require.Len(t, parent.Children, 1)
	require.Equal(t, int64(99), parent.Children[0].Id)
}

func TestConvertCategoryList_FlatProjection(t *testing.T) {
	tree := system.CategoryTree{
		Category: &system.Category{Id: 0, Pid: -1, Name: "root", Show: true},
		Children: []*system.CategoryTree{
			{
				Category: &system.Category{Id: 1, Pid: 0, Name: "电影", Show: true},
				Children: []*system.CategoryTree{
					{Category: &system.Category{Id: 6, Pid: 1, Name: "动作", Show: true}},
				},
			},
		},
	}
	flat := ConvertCategoryList(tree)
	require.Len(t, flat, 3)
	require.Equal(t, "root", flat[0].Name)
	require.Equal(t, "电影", flat[1].Name)
	require.Equal(t, "动作", flat[2].Name)
}

func TestGenFilmPlayList_FilterM3u8AndMp4(t *testing.T) {
	playUrl := "EP1$http://a.com/1.m3u8#EP2$http://a.com/2.m3u8$$$EP1$http://b.com/1.flv"
	got := GenFilmPlayList(playUrl, "$$$")
	require.Len(t, got, 1, "only the m3u8 source should pass the filter")
	require.Len(t, got[0], 2)
	require.Equal(t, "EP1", got[0][0].Episode)
	require.Equal(t, "http://a.com/1.m3u8", got[0][0].Link)
}

func TestGenFilmPlayList_NoSeparatorPath(t *testing.T) {
	got := GenFilmPlayList("EP1$http://x/y.mp4", "")
	require.Len(t, got, 1)
	require.Len(t, got[0], 1)
	require.Equal(t, "http://x/y.mp4", got[0][0].Link)

	// 非 m3u8/mp4 直接返回空
	require.Empty(t, GenFilmPlayList("EP1$http://x/y.flv", ""))
}

func TestConvertPlayUrl_HandlesMissingDollar(t *testing.T) {
	got := ConvertPlayUrl("EP1$http://x.com/1.m3u8#http://x.com/2.m3u8")
	require.Len(t, got, 2)
	require.Equal(t, "EP1", got[0].Episode)
	require.Equal(t, "(｀・ω・´)", got[1].Episode, "fallback episode for malformed item")
}

func TestConvertVirtualPicture_SkipsEmpty(t *testing.T) {
	got := ConvertVirtualPicture([]system.MovieDetail{
		{Id: 1, Picture: "p1"},
		{Id: 2, Picture: ""},
		{Id: 3, Picture: "p3"},
	})
	require.Len(t, got, 2)
	require.Equal(t, int64(1), got[0].Id)
	require.Equal(t, int64(3), got[1].Id)
}

func TestConvertFilmDetail_FieldsAndPlayList(t *testing.T) {
	in := collect.FilmDetail{
		VodID:       100,
		TypeID:      6,
		TypeID1:     1,
		VodName:     "影片",
		VodPlayURL:  "EP1$http://x/1.m3u8",
		VodPlayNote: "$$$",
	}
	out := ConvertFilmDetail(in)
	require.Equal(t, int64(100), out.Id)
	require.Equal(t, int64(6), out.Cid)
	require.Equal(t, int64(1), out.Pid)
	require.Equal(t, "影片", out.Name)
	require.Len(t, out.PlayList, 1)
}
