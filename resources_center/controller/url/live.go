package url

import (
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	urlService "xgit.xiaoniangao.cn/xngo/service/resources_center/service/url"
)

// LiveVideoReq ..
type LiveVideoReq []struct {
	ID string `json:"id" binding:"required"`
}

// LiveVideoResp ...
type LiveVideoResp struct {
	URLs map[string]RespURL `json:"urls"`
}

func LiveVideoURL(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req LiveVideoReq
	if !xc.GetReqObject(&req) {
		return
	}
	resp := &LiveVideoResp{URLs: make(map[string]RespURL)}

	for _, v := range req {
		url, urlInternal := urlService.GetLiveVideoURL(xc, v.ID)
		resp.URLs[v.ID] = RespURL{URL: url, URLInternal: urlInternal}
	}

	xc.ReplyOK(resp)
}

func LiveGuideVideoURL(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req LiveVideoReq
	if !xc.GetReqObject(&req) {
		return
	}
	resp := &LiveVideoResp{URLs: make(map[string]RespURL)}

	for _, v := range req {
		url, urlInternal := urlService.GetLiveGuideVideoURL(xc, v.ID)
		resp.URLs[v.ID] = RespURL{URL: url, URLInternal: urlInternal}
	}

	xc.ReplyOK(resp)
}
