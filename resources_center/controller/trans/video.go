package trans

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/resource"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/trans"
	urlService "xgit.xiaoniangao.cn/xngo/service/resources_center/service/url"
)

type ReqVideoTransMp4 struct {
	Key         string `json:"id"`
	Prod        int    `json:"prod"`
	Proj        string `json:"proj"`
	ProjectName string `json:"p_name"`
	TplID       string `json:"tpl_id"`
	Kind        int    `json:"type"`
	UserData    string `json:"user_data"`
}
type RespVideoTransMp4 struct {
	Status int    `json:"status"`
	JobID  string `json:"jobID"`
}

func checkTransTplID(tpl string) bool {
	switch tpl {
	case api.TransTemplate720PMP4:
		return true
	}
	return false
}
func checkTransRes(rates []api.RateInfo, tplID string) string {
	for _, rate := range rates {
		if rate.TplID == tplID {
			return rate.ID
		}
	}
	return ""
}
func VideoTransToMp4(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req ReqVideoTransMp4
	if !xc.GetReqObject(&req) {
		return
	}
	if req.Key == "" || req.ProjectName == "" || req.Proj == "" || req.Prod <= 0 || req.Kind <= 0 || checkTransTplID(req.TplID) == false {
		xc.ReplyFail(lib.CodePara)
		return
	}

	//防止重复转码
	var err error
	var resp RespVideoTransMp4
	var doc *api.XngResourceInfoDoc //防止重复请求转码
	if req.Kind != api.ResourceTypeGuideVideo {
		doc, err = resource.ByID(xc, req.Key)
		if err != nil {
			xlog.ErrorC(xc, "VideoTransToMp4.ByID failed, req:%v, err:%v", req, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
	}
	if doc != nil && doc.VRate != nil {
		transID := checkTransRes(doc.VRate, req.TplID)
		if transID != "" {
			var transDoc *api.XngResourceInfoDoc
			transDoc, err = resource.ByID(xc, transID)
			if err != nil || transDoc == nil {
				xlog.ErrorC(xc, "VideoTransToMp4.ByID search transID failed, req:%v, err:%v", req, err)
				xc.ReplyFail(lib.CodeSrv)
				return
			}
			var url, urlInternal string
			if transDoc.Cover != 0 {
				cover := fmt.Sprintf("%d", transDoc.Cover)
				url, urlInternal = urlService.GetImageURL(xc, cover, "imageMogr2/thumbnail/750x500/format/jpg")
			}
			resDoc := api.ResDocWithCoverUrl{
				ResId:            fmt.Sprintf("%d", transDoc.ResId),
				Type:             transDoc.Type,
				Size:             transDoc.Size,
				QeTag:            transDoc.QeTag,
				Upt:              transDoc.Upt,
				Fmt:              transDoc.Fmt,
				W:                transDoc.W,
				H:                transDoc.H,
				Du:               transDoc.Du,
				Cover:            fmt.Sprintf("%d", transDoc.Cover),
				Code:             transDoc.Code,
				Ort:              transDoc.Ort,
				CoverUrl:         url,
				CoverUrlInternal: urlInternal,
				UserData:         req.UserData,
			}
			err = resource.NotifyMediaInfoMq(xc, req.Prod, req.Proj, req.ProjectName, &resDoc, 1)
			if err != nil {
				xlog.ErrorC(xc, "VideoTransToMp4.NotifyMediaInfoMq failed, req:%v, err:%v", req, err)
				xc.ReplyFail(lib.CodeSrv)
				return
			}
			resp.Status = 2
			xc.ReplyOK(resp)
			return
		}
	}
	status, _, jobID, err := trans.VideoMp4Trans(xc, req.Key, req.Kind, req.Prod, req.Proj, req.TplID, req.UserData, req.ProjectName)
	if err != nil {
		xlog.ErrorC(xc, "VideoTransToMp4.VideoMp4Trans failed, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resp.Status = status
	resp.JobID = jobID
	xc.ReplyOK(resp)
}
