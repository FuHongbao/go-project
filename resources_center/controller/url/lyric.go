package url

import (
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	urlService "xgit.xiaoniangao.cn/xngo/service/resources_center/service/url"
)

type LyricReq []struct {
	ID string `json:"id" binding:"required"`
}

type LyricResp struct {
	URLs map[string]RespURL `json:"urls"`
}

// LyricURL
func LyricURL(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req LyricReq
	if !xc.GetReqObject(&req) {
		return
	}

	resp := &LyricResp{URLs: make(map[string]RespURL)}

	for _, v := range req {
		url, urlInternal := urlService.GetLyricURL(xc, v.ID)
		resp.URLs[v.ID] = RespURL{URL: url, URLInternal: urlInternal}
	}

	xc.ReplyOK(resp)
}
