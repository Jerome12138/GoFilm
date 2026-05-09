package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"server/config"
	"server/logic"
	"server/model/system"
)

// 从已鉴权 context 提取 user id; 失败则返回 false 并已写入响应
func currentUserID(c *gin.Context) (uint, bool) {
	v, ok := c.Get(config.AuthUserClaims)
	if !ok {
		system.Failed("用户身份信息缺失, 请先登录", c)
		return 0, false
	}
	uc, ok := v.(*system.UserClaims)
	if !ok || uc == nil {
		system.Failed("用户身份信息无效", c)
		return 0, false
	}
	return uc.UserID, true
}

func parsePage(c *gin.Context) *system.Page {
	current, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
	if current < 1 {
		current = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return &system.Page{Current: current, PageSize: pageSize}
}

// ============================== 观看历史 ==============================

// HistoryUpsert 上报/更新观看进度. body: HistoryUpsertParams
func HistoryUpsert(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	var p logic.HistoryUpsertParams
	if err := c.ShouldBindJSON(&p); err != nil {
		system.Failed("请求参数异常", c)
		return
	}
	if err := logic.UL.UpsertHistory(uid, p); err != nil {
		system.Failed(err.Error(), c)
		return
	}
	system.SuccessOnlyMsg("已记录观看历史", c)
}

// HistoryList 当前用户分页历史
func HistoryList(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	page := parsePage(c)
	list := logic.UL.ListHistory(uid, page)
	system.Success(gin.H{"list": list, "page": page}, "获取观看历史成功", c)
}

// HistoryDelete 删除单条历史: 优先 ?id=, 兜底 ?mid=
func HistoryDelete(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	idVal, _ := strconv.ParseUint(c.Query("id"), 10, 64)
	midVal, _ := strconv.ParseInt(c.Query("mid"), 10, 64)
	if err := logic.UL.DeleteHistory(uid, uint(idVal), midVal); err != nil {
		system.Failed(err.Error(), c)
		return
	}
	system.SuccessOnlyMsg("删除成功", c)
}

// HistoryClear 清空当前用户全部历史
func HistoryClear(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	if err := logic.UL.ClearHistory(uid); err != nil {
		system.Failed(err.Error(), c)
		return
	}
	system.SuccessOnlyMsg("观看历史已清空", c)
}

// ============================== 收藏 ==============================

// FavoriteAdd 添加收藏 (幂等)
func FavoriteAdd(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	var p logic.FavoriteParams
	if err := c.ShouldBindJSON(&p); err != nil {
		system.Failed("请求参数异常", c)
		return
	}
	if err := logic.UL.AddFavorite(uid, p); err != nil {
		system.Failed(err.Error(), c)
		return
	}
	system.SuccessOnlyMsg("已添加到收藏", c)
}

// FavoriteRemove 取消收藏: ?mid=
func FavoriteRemove(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	mid, _ := strconv.ParseInt(c.Query("mid"), 10, 64)
	if err := logic.UL.RemoveFavorite(uid, mid); err != nil {
		system.Failed(err.Error(), c)
		return
	}
	system.SuccessOnlyMsg("已取消收藏", c)
}

// FavoriteList 收藏分页
func FavoriteList(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	page := parsePage(c)
	list := logic.UL.ListFavorite(uid, page)
	system.Success(gin.H{"list": list, "page": page}, "获取收藏列表成功", c)
}

// FavoriteCheck 是否已收藏: ?mid=  → { favorited: bool }
func FavoriteCheck(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	mid, _ := strconv.ParseInt(c.Query("mid"), 10, 64)
	if mid <= 0 {
		system.Failed("缺少 mid", c)
		return
	}
	system.Success(gin.H{"favorited": logic.UL.CheckFavorite(uid, mid)}, "ok", c)
}
