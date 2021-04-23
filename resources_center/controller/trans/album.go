package trans

import (
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/resource"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/trans"
)

type ReqAlbumTransCode struct {
	Key  string `json:"id"`
	Prod int    `json:"prod"`
	Proj string `json:"proj"`
	//W      int  		`json:"w"`
	//H	   int 			`json:"h"`
	UserData string `json:"user_data"`
}
type RespAlbumTransCode struct {
	Status int    `json:"status"`
	JobID  string `json:"jobID"`
}

func AlbumTransToMp4(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req ReqAlbumTransCode
	if !xc.GetReqObject(&req) {
		return
	}
	var err error
	var qdoc *api.XngResourceInfoDoc //防止回调失败重试，数据库重复插入
	qdoc, err = resource.ByID(xc, req.Key)
	if err != nil {
		xlog.ErrorC(xc, "AlbumTransToMp4.ByID failed, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	var resp RespAlbumTransCode
	if qdoc != nil {
		resp.Status = 2
		xc.ReplyOK(resp)
		return
	}
	status, jobID, err := trans.AlbumM3u8Trans(xc, req.Key, api.ResourceTypeAlbum, req.Prod, req.Proj, req.UserData)
	if err != nil {
		xlog.ErrorC(xc, "AlbumTransToMp4.AlbumM3u8Trans failed, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resp.Status = status
	resp.JobID = jobID
	xc.ReplyOK(resp)
}
