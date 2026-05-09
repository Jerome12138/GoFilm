package system

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"server/config"
	"server/plugin/db"
)

func TestSaveDetails_PipelineWritesDetailAndBasicInfo(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()

	list := []MovieDetail{
		{Id: 1, Cid: 6, Pid: 1, Name: "A", Picture: "pa.png",
			MovieDescriptor: MovieDescriptor{Year: "2024", State: "正片", Remarks: "HD"}},
		{Id: 2, Cid: 6, Pid: 1, Name: "B", Picture: "pb.png",
			MovieDescriptor: MovieDescriptor{Year: "2023"}},
	}
	require.NoError(t, SaveDetails(list))

	for _, d := range list {
		// detail key
		raw, err := db.Rdb.Get(db.Cxt, fmt.Sprintf(config.MovieDetailKey, d.Cid, d.Id)).Result()
		require.NoError(t, err)
		var got MovieDetail
		require.NoError(t, json.Unmarshal([]byte(raw), &got))
		require.Equal(t, d.Name, got.Name)

		// basic info key
		braw, err := db.Rdb.Get(db.Cxt, fmt.Sprintf(config.MovieBasicInfoKey, d.Cid, d.Id)).Result()
		require.NoError(t, err)
		var basic MovieBasicInfo
		require.NoError(t, json.Unmarshal([]byte(braw), &basic))
		require.Equal(t, d.Name, basic.Name)
		require.Equal(t, d.Picture, basic.Picture)
	}
}

func TestSaveDetails_EmptyListNoop(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()
	require.NoError(t, SaveDetails(nil))
	keys, _ := db.Rdb.Keys(db.Cxt, "*").Result()
	require.Equal(t, 0, len(keys), "no key should be written for empty input")
}

func TestGetBasicInfoBySearchInfos_EmptyShortCircuit(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()
	got := GetBasicInfoBySearchInfos()
	require.Nil(t, got)
}

func TestMgetBasicInfo_OrderAndMissing(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()
	require.NoError(t, SaveDetails([]MovieDetail{
		{Id: 1, Cid: 6, Name: "A"},
		{Id: 2, Cid: 6, Name: "B"},
	}))
	got := mgetBasicInfo([]string{
		fmt.Sprintf(config.MovieBasicInfoKey, 6, 1),
		fmt.Sprintf(config.MovieBasicInfoKey, 6, 999), // missing
		fmt.Sprintf(config.MovieBasicInfoKey, 6, 2),
	})
	require.Len(t, got, 2)
	require.Equal(t, "A", got[0].Name)
	require.Equal(t, "B", got[1].Name)
}

func TestGetBasicInfoBySearchInfos_RoundTrip(t *testing.T) {
	_, cleanup := withMiniRedis(t)
	defer cleanup()
	require.NoError(t, SaveDetails([]MovieDetail{
		{Id: 1, Cid: 6, Name: "A"},
		{Id: 2, Cid: 6, Name: "B"},
	}))
	infos := []SearchInfo{{Mid: 1, Cid: 6}, {Mid: 2, Cid: 6}}
	got := GetBasicInfoBySearchInfos(infos...)
	require.Len(t, got, 2)
	require.Equal(t, "A", got[0].Name)
	require.Equal(t, "B", got[1].Name)
}
