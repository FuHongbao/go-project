package resource

import (
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/resource"
)

type ReqCheckResExists struct {
	Kind int    `json:"type"`
	Qid  string `json:"id"`
}
type RespCheckResExists struct {
	IsExist bool `json:"is_exist"`
}

func CheckResExists(c *gin.Context) {
	xc := xng.NewXContext(c)
	req := ReqCheckResExists{}
	if !xc.GetReqObject(&req) {
		return
	}
	if req.Qid == "" || req.Kind <= 0 {
		xc.ReplyFail(lib.CodePara)
		return
	}
	isExist, err := resource.CheckResExists(xc, req.Qid, req.Kind)
	if err != nil {
		xlog.ErrorC(xc, "CheckResExists.CheckResExists failed, req:[%v], err:[%v]", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resp := &RespCheckResExists{IsExist: isExist}
	xc.ReplyOK(resp)
	return
}
