package url

import (
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	urlService "xgit.xiaoniangao.cn/xngo/service/resources_center/service/url"
)

// MusicReq ..
type MusicReq []struct {
	ID string `json:"id" binding:"required"`
}

// MusicResp
type MusicResp struct {
	URLs map[string]RespURL `json:"urls"`
}

func AuditMusicURL(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req MusicReq
	if !xc.GetReqObject(&req) {
		return
	}
	resp := &MusicResp{URLs: make(map[string]RespURL)}
	for _, v := range req {
		url, urlInternal := urlService.GetAuditMusicURL(xc, v.ID)
		resp.URLs[v.ID] = RespURL{URL: url, URLInternal: urlInternal}
	}

	xc.ReplyOK(resp)
}

func MusicURL(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req MusicReq
	if !xc.GetReqObject(&req) {
		return
	}
	resp := &MusicResp{URLs: make(map[string]RespURL)}
	for _, v := range req {
		url, urlInternal := urlService.GetMusicURL(xc, v.ID)
		resp.URLs[v.ID] = RespURL{URL: url, URLInternal: urlInternal}
	}

	xc.ReplyOK(resp)
}
