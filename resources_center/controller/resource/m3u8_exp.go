package resource

import (
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	resourceDao "xgit.xiaoniangao.cn/xngo/service/resources_center/dao/resource"
)

type ReqSetM3u8ExpID struct {
	IDs []string `json:"ids"`
}

func SetM3u8ExpID(c *gin.Context) {
	xc := xng.NewXContext(c)
	req := ReqSetM3u8ExpID{}
	if !xc.GetReqObject(&req) {
		return
	}
	if len(req.IDs) <= 0 {
		xc.ReplyFail(lib.CodePara)
		return
	}
	err := resourceDao.SetM3u8ExpID(xc, req.IDs)
	if err != nil {
		xlog.ErrorC(xc, "SetM3u8ExpID.SetM3u8ExpID failed, err:[%v], req:[%v]", err, req)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.ReplyOK(nil)
}
