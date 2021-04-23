package resource

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/resource"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils"
)

type ReqByID struct {
	ID string `json:"id" binding:"required"`
}

type Resp struct {
	ResId     string  `json:"id"`
	Type      int     `json:"ty"`
	Size      int64   `json:"size"`
	QeTag     string  `json:"qetag"`
	Upt       int64   `json:"upt"`
	Fmt       string  `json:"fmt"`
	W         int     `json:"w"`
	H         int     `json:"h"`
	Du        float64 `json:"du,omitempty"`
	Cover     string  `json:"cover,omitempty"`
	Code      string  `json:"code,omitempty"`
	Ort       int     `json:"ort,omitempty"`
	TransCode *int    `json:"trans,omitempty" bson:"trans"`
}

// ByID ...
func ByID(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req ReqByID
	if !xc.GetReqObject(&req) {
		return
	}

	resp, err := resource.ByID(xc, req.ID)
	if err != nil {
		xlog.ErrorC(xc, "fail to get doc by id, id:%s, err:%v", req.ID, err)
		xc.ReplyFail(lib.CodeSrv)
	}
	if resp == nil {
		xc.ReplyFail(lib.CodeNotExist)
	} else {
		ty := resp.Type
		if ty == api.ResourceTypeGroupImg {
			ty = api.ResourceTypeImg
		}
		transCode := resp.TransCode
		if transCode == nil {
			transCode = &resource.ResStatusTrans
		}
		ret := Resp{
			ResId:     fmt.Sprintf("%d", resp.ResId),
			Type:      ty,
			Size:      resp.Size,
			QeTag:     resp.QeTag,
			Upt:       resp.Upt,
			Fmt:       resp.Fmt,
			W:         resp.W,
			H:         resp.H,
			Du:        resp.Du,
			Cover:     fmt.Sprintf("%d", resp.Cover),
			Code:      resp.Code,
			Ort:       resp.Ort,
			TransCode: resp.TransCode,
		}
		xc.ReplyOK(ret)
	}
}

type ReqGetByIDBatch struct {
	IDs []string `json:"ids"`
}
type RespGetByIDBatch struct {
	Docs    map[string]*Resp `json:"docs"`
	FailIDs []string         `json:"fail_ids"`
}

// 跟进id批量获取资源信息
func ByIDBatch(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req ReqGetByIDBatch
	if !xc.GetReqObject(&req) {
		return
	}
	if len(req.IDs) <= 0 {
		xc.ReplyFail(lib.CodePara)
		return
	}
	var ids []string
	for _, id := range req.IDs {
		if id != "" {
			ids = append(ids, id)
		}
	}
	docs, idsNotFound, err := resource.ByIDBatch(xc, ids)
	if err != nil {
		xlog.ErrorC(xc, "fail to get docs by id, ids:[%v], err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resp := RespGetByIDBatch{
		Docs:    make(map[string]*Resp),
		FailIDs: idsNotFound,
	}
	//resp.FailIDs = idsNotFound
	if docs != nil {
		for id, doc := range docs {
			ty := doc.Type
			if ty == api.ResourceTypeGroupImg {
				ty = api.ResourceTypeImg
			}
			transCode := doc.TransCode
			if transCode == nil {
				transCode = &resource.ResStatusTrans
			}
			ret := &Resp{
				ResId:     id,
				Type:      ty,
				Size:      doc.Size,
				QeTag:     doc.QeTag,
				Upt:       doc.Upt,
				Fmt:       doc.Fmt,
				W:         doc.W,
				H:         doc.H,
				Du:        doc.Du,
				Cover:     fmt.Sprintf("%d", doc.Cover),
				Code:      doc.Code,
				Ort:       doc.Ort,
				TransCode: transCode,
			}
			resp.Docs[id] = ret
		}
	}

	xc.ReplyOK(resp)
	return
}

//获取上传状态请求结构体
type ReqByEtag struct {
	QeTag string `json:"qetag" binding:"gt=8"`
	Type  int    `json:"ty"` //资源类型
}

func ByEtag(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req ReqByEtag
	if !xc.GetReqObject(&req) {
		return
	}
	req.QeTag = utils.GetQetagByResType(req.Type, req.QeTag)
	qid, err := resource.ByQeTag(xc, req.QeTag)
	if err != nil {
		xlog.ErrorC(xc, "failed to get qid by qetag, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if qid == "" {
		xc.ReplyFail(lib.CodeNotExist)
		return
	}
	doc, err := resource.ByID(xc, qid)
	if err != nil {
		xlog.ErrorC(xc, "failed to get media info by qid, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if doc != nil {
		if doc.Type == 10 && (doc.Cover >= 0 || doc.Fmt == "mov,mp4,m4a,3gp,3g2,mj2," || doc.W == 0 || doc.H == 0) { //排除xbd type为10得资源得干扰
			xc.ReplyFail(lib.CodeNotExist)
			return
		}
		ty := doc.Type
		if ty == api.ResourceTypeGroupImg {
			ty = api.ResourceTypeImg
		}
		transCode := doc.TransCode
		if transCode == nil {
			transCode = &resource.ResStatusTrans
		}
		ret := Resp{
			ResId:     fmt.Sprintf("%d", doc.ResId),
			Type:      ty,
			Size:      doc.Size,
			QeTag:     doc.QeTag,
			Upt:       doc.Upt,
			Fmt:       doc.Fmt,
			W:         doc.W,
			H:         doc.H,
			Du:        doc.Du,
			Cover:     fmt.Sprintf("%d", doc.Cover),
			Code:      doc.Code,
			Ort:       doc.Ort,
			TransCode: transCode,
		}
		xc.ReplyOK(ret)
	} else {
		xc.ReplyFail(lib.CodeNotExist)
	}
	//resp, err := videoService.GetResDocByQeTag(req.QeTag)
	//if err != nil {
	//	xlog.ErrorC(xc, "failed to GetResDocByQeTag, req:%v, err:%v", req, err)
	//	xc.ReplyFail(lib.CodeSrv)
	//	return
	//}

	//if resp == nil {
	//	xc.ReplyFail(lib.CodeNotExist)
	//} else {
	//	ret := Resp{
	//		ResId: fmt.Sprintf("%d", resp.ResId),
	//		Type:  resp.Type,
	//		Size:  resp.Size,
	//		QeTag: resp.QeTag,
	//		Upt:   resp.Upt,
	//		Fmt:   resp.Fmt,
	//		W:     resp.W,
	//		H:     resp.H,
	//		Du:    resp.Du,
	//		Cover: fmt.Sprintf("%d", resp.Cover),
	//		Code:  resp.Code,
	//	}
	//	xc.ReplyOK(ret)
	//}
}
