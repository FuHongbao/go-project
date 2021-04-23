package snap

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByQid"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/resource"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/snap"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/sts"
)

type ReqAlbumSnap struct {
	Project   string  `json:"pj"`
	Key       string  `json:"key"`
	Du        float64 `json:"du"`
	Code      string  `json:"code"`
	StartTime float64 `json:"start_time"`
}
type RespAlbumSnap struct {
	ResId string `json:"id"`
	Type  int    `json:"ty"`
	Size  int64  `json:"size"`
	Upt   int64  `json:"upt"`
	Fmt   string `json:"fmt"`
	W     int    `json:"w"`
	H     int    `json:"h"`
	Ort   int    `json:"ort,omitempty"`
}

func AlbumSnapShot(c *gin.Context) {
	xc := xng.NewXContext(c)

	var req ReqAlbumSnap
	if !xc.GetReqObject(&req) {
		return
	}
	if req.Project == "" || req.Code == "" || req.Du <= api.SnapConfShortTime || req.StartTime <= api.SnapConfShortTime || req.Key == "" {
		xc.ReplyFail(lib.CodePara)
		return
	}
	idCh := make(chan int64, 1)
	go resource.GetDistributedId(idCh)
	stsData, err := sts.GetUpToken(xc, api.StsForMtsAlbum)
	if err != nil {
		xlog.ErrorC(xc, "AlbumSnapShot get sts information err:%v, req:%v", err, req)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if stsData == nil {
		xlog.ErrorC(xc, "AlbumSnapShot get sts information err, sts data is nil")
		return
	}
	snapId := <-idCh
	if snapId <= 0 {
		xlog.ErrorC(xc, "AlbumSnapShot get snap id err, id <= 0")
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	//对资源进行截图
	srcBucket, destBucket := stsData.Bucket, conf.C.Bucket.Resource
	err = resource.GetSnapShot(xc, req.Key, stsData, snapId, req.Du, req.Code, srcBucket, destBucket, req.StartTime)
	if err != nil {
		xlog.ErrorC(xc, "AlbumSnapShot failed to get snap shot, err:%v, req:%v", err, req)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	//获取截图信息并整理doc
	snapDoc, err := resource.OrganizeSnapShotDoc(xc, req.Project, snapId)
	if err != nil {
		xlog.ErrorC(xc, "AlbumSnapShot failed to get snap shot information, err:%v, req:%v", err, req)
		xc.ReplyFail(lib.CodeSrv)
		return
	}

	//snap doc 存库
	err = DaoByQid.InsertResourceDoc(snapId, snapDoc)
	if err != nil {
		xlog.ErrorC(xc, "AlbumSnapShot failed to insert snap doc to DB, err:%v, req:%v", err, req)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resp := RespAlbumSnap{
		ResId: fmt.Sprintf("%d", snapDoc.ResId),
		Type:  snapDoc.Type,
		Size:  snapDoc.Size,
		Upt:   snapDoc.Upt,
		Fmt:   snapDoc.Fmt,
		W:     snapDoc.W,
		H:     snapDoc.H,
		Ort:   snapDoc.Ort,
	}
	xc.ReplyOK(resp)
}

type ReqReplaceSnap struct {
	Key   string `json:"key"`
	Cover string `json:"cover"`
}
type RespReplaceSnap struct {
	Status int `json:"status"`
}

func ReplaceSnapShot(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req ReqReplaceSnap
	if !xc.GetReqObject(&req) {
		return
	}
	if req.Cover == "" || req.Key == "" {
		xc.ReplyFail(lib.CodePara)
		return
	}
	resID, err := strconv.ParseInt(req.Key, 10, 64)
	if err != nil {
		xlog.ErrorC(xc, "ReplaceSnapShot.ParseInt failed, err:[%v], req:[v]", err, req)
		xc.ReplyFail(lib.CodePara)
		return
	}
	snapID, err := strconv.ParseInt(req.Cover, 10, 64)
	if err != nil {
		xlog.ErrorC(xc, "ReplaceSnapShot.ParseInt failed, err:[%v], req:[v]", err, req)
		xc.ReplyFail(lib.CodePara)
		return
	}
	ret, err := snap.UpdateResCover(xc, resID, snapID)
	if err != nil {
		xlog.ErrorC(xc, "ReplaceSnapShot.UpdateResCover failed, err:[%v], req:[v]", err, req)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	status := 0
	if ret == true {
		status = 1
	}
	resp := &RespReplaceSnap{Status: status}
	xc.ReplyOK(resp)
}

type ReqVideoFrameList struct {
	Key       string `json:"key"`
	Du        int64  `json:"du"` //视频时长，单位毫秒
	W         int    `json:"w"`
	H         int    `json:"h"`
	StartTime int64  `json:"start_time"` //开始截图时间，单位毫秒
	Cnt       int    `json:"cnt"`        //截图数
	SpaceTime int64  `json:"space_time"` //间隔时间，单位毫秒
}
type RespVideoFrameList struct {
	Id   string   `json:"qid"`
	Urls []string `json:"frame_urls"`
}

func VideoFrameList(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req ReqVideoFrameList
	if !xc.GetReqObject(&req) {
		return
	}
	if req.Key == "" || req.Du <= 0 || req.Cnt <= 0 || req.StartTime < 0 || req.StartTime >= req.Du {
		xc.ReplyFail(lib.CodePara)
		return
	}
	frameCnt := req.Cnt
	temp := int(req.Du-req.StartTime)/req.Cnt + 1
	if frameCnt > temp {
		frameCnt = temp
	}
	urls := snap.GetVideoFrameUrls(xc, req.Key, frameCnt, req.StartTime, req.SpaceTime, req.W, req.H)
	resp := &RespVideoFrameList{
		Id:   req.Key,
		Urls: urls,
	}
	xc.ReplyOK(resp)
}

type ReqAlbumFrameList struct {
	Key       string `json:"key"`
	Du        int64  `json:"du"` //视频时长，单位毫秒
	W         int    `json:"w"`
	H         int    `json:"h"`
	StartTime int64  `json:"start_time"` //开始截图时间，单位毫秒
	Cnt       int    `json:"cnt"`        //截图数
	SpaceTime int64  `json:"space_time"` //间隔时间，单位毫秒
}
type RespAlbumFrameUrls struct {
	Urls     []string `json:"frame_urls"`
	ErrorMsg string   `json:"err_msg,omitempty"`
}
type RespAlbumFrameList struct {
	URLs map[string]RespAlbumFrameUrls `json:"urls"`
}

func checkParamAlbumFrameList(req ReqAlbumFrameList) string {
	if req.Key == "" {
		return "资源key字段验证不通过"
	}
	if req.Cnt <= 0 {
		return "截帧数cnt字段验证不通过"
	}
	if req.StartTime > req.Du {
		return "截帧起始时间不能大于视频总时长"
	}
	return ""
}

func AlbumFrameList(c *gin.Context) {
	xc := xng.NewXContext(c)
	var reqs []ReqAlbumFrameList
	if !xc.GetReqObject(&reqs) {
		return
	}
	if len(reqs) <= 0 {
		xc.ReplyFail(lib.CodePara)
		return
	}
	resp := &RespAlbumFrameList{URLs: make(map[string]RespAlbumFrameUrls)}
	for _, req := range reqs {
		msg := checkParamAlbumFrameList(req)
		if msg != "" {
			resp.URLs[req.Key] = RespAlbumFrameUrls{ErrorMsg: msg}
			continue
		}
		frameCnt := req.Cnt
		temp := int(req.Du-req.StartTime)/req.Cnt + 1
		if frameCnt > temp {
			frameCnt = temp
		}
		urls := snap.GetAlbumFrameUrls(xc, req.Key, frameCnt, req.StartTime, req.SpaceTime, req.W, req.H)
		resp.URLs[req.Key] = RespAlbumFrameUrls{Urls: urls}
	}
	xc.ReplyOK(resp)
}
