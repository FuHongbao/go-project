package url

import (
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	urlService "xgit.xiaoniangao.cn/xngo/service/resources_center/service/url"
)

type AppLogReq []struct {
	ID string `json:"id"`
}
type AppLogResp struct {
	URLs map[string]RespURL `json:"urls"`
}

func AppLogURL(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req AppLogReq
	if !xc.GetReqObject(&req) {
		return
	}
	resp := &AppLogResp{URLs: make(map[string]RespURL)}

	for _, v := range req {
		url, urlInternal := urlService.GetAppLogURL(xc, v.ID)
		resp.URLs[v.ID] = RespURL{URL: url, URLInternal: urlInternal}
	}
	xc.ReplyOK(resp)
}
