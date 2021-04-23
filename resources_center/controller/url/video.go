package url

import (
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	urlService "xgit.xiaoniangao.cn/xngo/service/resources_center/service/url"
)

// VideoReq ..
type VideoReq []struct {
	ID string `json:"id" binding:"required"`
}

// VideoResp
type VideoResp struct {
	URLs map[string]RespURL `json:"urls"`
}

// VideoURL
func VideoURL(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req VideoReq
	if !xc.GetReqObject(&req) {
		return
	}

	resp := &VideoResp{URLs: make(map[string]RespURL)}

	for _, v := range req {
		url, urlInternal := urlService.GetVideoURL(xc, v.ID)
		resp.URLs[v.ID] = RespURL{URL: url, URLInternal: urlInternal}
	}

	xc.ReplyOK(resp)
}
