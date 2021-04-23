package sts

import (
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/sts"
)

type GetReq struct {
	Name string `json:"name"`
}

func Get(c *gin.Context) {
	xc := xng.NewXContext(c)

	var req GetReq
	if !xc.GetReqObject(&req) {
		return
	}

	token, err := sts.GetUpToken(xc, req.Name)
	if err != nil {
		xlog.ErrorC(xc, "get upload token err:%v", err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}

	xc.ReplyOK(token)
}
