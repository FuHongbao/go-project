package video

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByMid"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByQid"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByTag"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/videoService"
)

func NewAppUploadCallback(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req api.AppUploadCallbackReq
	if !xc.GetReqObject(&req) {
		return
	}
	if req.QeTag == "" || req.Qid <= 0 || req.FileType != api.ResourceTypeVideo || req.Mid <= 0 { //暂时只支持视频类型资源
		xc.ReplyFail(lib.CodePara)
		return
	}
	//1. 检查资源记录和用户资源记录是否已经存在
	qidKey := strconv.FormatInt(req.Qid, 10)
	doc, err := videoService.ByID(xc, qidKey)
	if err != nil {
		xlog.ErrorC(xc, "fail to get doc by id, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	//资源存在则处理user doc，返回资源信息
	if doc != nil {
		mDoc, err := DaoByMid.GetDocByMid(req.Mid, req.Qid)
		if err != nil {
			xlog.ErrorC(xc, "fail to get user doc by mid, req:%v, err:%v", req, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		if mDoc != nil { //user doc存在则 更新upt和d字段
			err = videoService.UpdateUserResDoc(mDoc, req.Mid)
			if err != nil {
				xlog.ErrorC(xc, "fail to update user doc, req:%v, err:%v", req, err)
				xc.ReplyFail(lib.CodeSrv)
				return
			}
		} else { //user doc 不存在则 1.生成doc 2.更新资源引用计数 3.存入user资源库
			mDoc, _, err := videoService.CreateNewUserResDoc(doc, req.Mid)
			if err != nil {
				xlog.ErrorC(xc, "fail to create user doc, req:%v, err:%v", req, err)
				xc.ReplyFail(lib.CodeSrv)
				return
			}
			err = DaoByQid.UpdateResourceRef(req.Qid)
			if err != nil {
				xlog.ErrorC(xc, "failed to update resource ref, req:%v, err:%v", req, err)
				xc.ReplyFail(lib.CodeSrv)
				return
			}
			err = videoService.AddUserResDoc(mDoc, req.Mid)
			if err != nil {
				xlog.ErrorC(xc, "failed to add user resource doc, req:%v, err:%v", req, err)
				xc.ReplyFail(lib.CodeSrv)
				return
			}
		}
		resp := videoService.GetCallbackInfo(doc, mDoc.ResId)
		xc.ReplyOK(resp)
		return
	}
	//新视频资源进行信息获取和截图
	stsData, err := videoService.GetStsForMts()
	if err != nil {
		xlog.ErrorC(xc, "get sts information error, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if stsData == nil {
		xlog.ErrorC(xc, "get sts information error, sts data is nil")
		xc.ReplyFail(lib.CodeSrv)
		return
	}

	ch := make(chan int64, 1)
	go videoService.GetDistributedId(ch)
	//获取资源信息
	doc, err = videoService.GetVideoResBaseDoc(xc, stsData, qidKey, req.QeTag)
	if err != nil {
		xlog.ErrorC(xc, "get media information error, err:%v", err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	doc.TransCode = &videoService.ResStatusTrans //app自己进行转码，不使用大文件转码功能
	snapId := <-ch
	if snapId <= 0 {
		xlog.ErrorC(xc, "get snap DistributedId failed")
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	doc.Cover = snapId
	doc.CoverTp = videoService.VideoResCoverType
	//进行截图及存储 1. 截图 2. 获取截图信息 3. 截图doc存储
	go func() {
		errIgnore := videoService.SubmitSnapShotFrame(stsData, req.Qid, snapId, doc.Du)
		if errIgnore != nil {
			xlog.ErrorC(xc, "failed to submit resource snap shot, snapId:%d, err:%v", snapId, errIgnore)
			ch <- req.Qid
			return
		}
		targetImageName := strconv.FormatInt(snapId, 10)
		snapDoc, errIgnore := videoService.GetImgInfo(xc, targetImageName, api.ResourceTypeImg)
		if errIgnore != nil {
			xlog.ErrorC(xc, "get snap image doc error:%v, req:%v", err, req)
			ch <- req.Qid
			return
		}
		if snapDoc == nil {
			xlog.ErrorC(xc, "get snap image doc error, doc is nil")
			ch <- req.Qid
			return
		}
		snapDoc.Src = api.UploadFromApp
		errIgnore = DaoByQid.InsertResourceDoc(snapId, snapDoc)
		if errIgnore != nil {
			xlog.ErrorC(xc, "insert snap image doc to DB error, err:%v", err)
			ch <- req.Qid
			return
		}
		ch <- req.Qid
	}()

	//存储doc:  1. 存储资源doc 2. 存储tag库doc 3. 获取user doc并存储
	err = DaoByQid.InsertResourceDoc(doc.ResId, doc)
	if err != nil {
		xlog.ErrorC(xc, "insert media doc to DB error, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = DaoByTag.AddXngTagDoc(req.QeTag, doc.ResId)
	if err != nil {
		xlog.ErrorC(xc, "fail to add tag doc, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	mDoc, _, err := videoService.CreateNewUserResDoc(doc, req.Mid)
	if err != nil {
		xlog.ErrorC(xc, "fail to create user doc, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = videoService.AddUserResDoc(mDoc, req.Mid)
	if err != nil {
		xlog.ErrorC(xc, "fail to insert user doc to DB, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	<-ch
	//  1. 拷贝资源到xphoto-ali   2. 设置content_type  3. 删除临时桶内原资源
	err = videoService.CopyVideoResToBucket(qidKey, stsData.Data.InputBucket, conf.C.Bucket.Resource, stsData, doc.Size)
	if err != nil {
		xlog.ErrorC(xc, "fail to copy resource to resource bucket, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = videoService.SetMetaContentType(qidKey, conf.C.Bucket.Resource, stsData, api.ContentTypeVideo)
	if err != nil {
		xlog.Error("set resource content-type error, err:%v", err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = videoService.DelResFromBucket(qidKey, stsData.Data.InputBucket, stsData)
	if err != nil {
		xlog.ErrorC(xc, "fail to del resource from upload bucket, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resp := videoService.GetCallbackInfo(doc, mDoc.ResId)
	xc.ReplyOK(resp)
}
