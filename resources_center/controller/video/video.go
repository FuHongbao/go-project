package video

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo/bson"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByQid"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/resource"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/videoService"
)

// GetVideoInfo godoc
// @Summary 获取资源信息接口（大文件上传）
// @Description 大文件上传完成后，客户端调用此接口获取资源信息（目前支持视频类型），进行封面展示
// @Accept  json
// @Produce  json
// @Param   body body api.MediaInfoReq true "请求相关的参数"
// @success 200 {object} api.MediaInfoResp "返回JSON数据"
// @Router /video/videoinfo [post]
func GetVideoInfo(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req api.MediaInfoReq
	if !xc.GetReqObject(&req) {
		return
	}
	if req.Mid <= 0 || req.QeTag == "" {
		xc.ReplyFail(lib.CodePara)
		return
	}
	ty := api.ResourceTypeVideo
	resp, err := videoService.MediaInfo(&req, ty)
	if err != nil || resp == nil {
		xlog.ErrorC(xc, "failed to get video info, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.ReplyOK(resp)
}

// SubmitVideoTrans godoc
// @Summary 提交转码接口
// @Description 大文件上传完成后，调用此接口进行转码压缩
// @Accept  json
// @Produce  json
// @Param   body body api.TransCodeReq true "请求相关的参数"
// @success 200 {object} api.TransCodeResp "返回JSON数据"
// @Router /video/transvideo [post]
func SubmitVideoTrans(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req api.TransCodeReq
	if !xc.GetReqObject(&req) {
		return
	}
	if req.Qid <= 0 {
		xc.ReplyFail(lib.CodePara)
		return
	}
	ty := api.ResourceTypeVideo
	status, err := videoService.SubmitTransCode(&req, ty)
	if err != nil {
		xlog.ErrorC(xc, "failed to submit video transcode, qid:%d", req.Qid)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.ReplyOK(api.TransCodeResp{Status: status})
}

// ResultCallBack godoc
// @Summary 阿里云转码消息回调接口
// @Description 阿里云转码结果回调到此接口（包含转码结果和对应qid）
// @Accept  json
// @Produce  json
// @Param   body body api.MNSMessage true "请求相关的参数"
// @success 204 {object} nil "返回数据：无"
// @Router /video/callback [post]
func ResultCallBack(c *gin.Context) {
	xc := xng.NewXContext(c)
	var message api.MNSMessageData
	if conf.Env != lib.PROD {
		var req api.MNSMessage
		if !xc.GetReqObjectFromXml(&req) {
			return
		}
		err := json.Unmarshal(req.Message, &message)
		if err != nil {
			xlog.ErrorC(xc, "unmarshal aliyun message failed, message:%s", string(req.Message))
			return
		}
	} else {
		if !xc.GetReqObject(&message) {
			return
		}
	}
	if message.UserData == "" {
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	//conf.Logger.Info("aliyun request = %v", message)
	if message.JobID == "" || message.State == "" || message.Type == "" {
		xlog.ErrorC(xc, "failed to deal aliyun callback message, error: Parameter error, request=%v", message)
		xc.Reply(http.StatusInternalServerError, nil)
		return
	}
	err := videoService.DealCallBack(&message)
	if err != nil {
		xlog.ErrorC(xc, "failed to deal aliyun callback message, err:%v, aliRequest:%v", message, err)
		//xc.Status(500)
		xc.Reply(http.StatusInternalServerError, nil)
		return
	}
	//FIXME：这个上面抛出了err也返回吗？ 那不是告诉ali成功了吗？ 怎么告诉它失败，然后它能重试ne ?
	//xc.Status(204)
	xc.Reply(http.StatusNoContent, nil)
}

// GetUploadStatus godoc
// @Summary 获取资源上传状态接口
// @Description 客户端调用此接口获取资源上传状态（已上传：2；未上传：0）
// @Accept  json
// @Produce  json
// @Param   body body api.UploadStatusReq true "请求相关的参数"
// @success 200 {object} api.UploadStatusResp "返回JSON数据"
// @Router /video/get_upload_status [post]
func GetUploadStatus(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req api.UploadStatusReq
	if !xc.GetReqObject(&req) {
		return
	}
	if req.QeTag == "" {
		xc.ReplyFail(lib.CodePara)
		return
	}
	status, err := videoService.GetUploadStatus(&req)
	if err != nil {
		xlog.ErrorC(xc, "failed to get upload status, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.ReplyOK(&api.UploadStatusResp{Status: status})
}

// SetUploadStatus godoc
// @Summary 直传成功通知接口
// @Description 客户端直传成功后调用此接口通知，业务后端进行资源信息和截图的获取及存储
// @Accept  json
// @Produce  json
// @Param   body body api.SetUploadStatusReq true "请求相关的参数"
// @success 200 {object} strut{} "返回数据：无"
// @Router /video/set_upload_status [post]
func SetUploadStatus(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req api.SetUploadStatusReq
	if !xc.GetReqObject(&req) {
		return
	}
	if req.QeTag == "" || req.Qid <= 0 { //FIXME:req.Qid <= 0
		xc.ReplyFail(lib.CodePara)
		return
	}
	ty := api.ResourceTypeVideo
	err := videoService.SetUploadStatus(&req, ty)
	if err != nil {
		xlog.ErrorC(xc, "failed to set upload status, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.ReplyOK(struct{}{})
}

// GetTempVoucher godoc
// @Summary 获取上传凭证接口
// @Description 客户端获取临时上传凭证（sts信息与目标bucket等），进行前端直传
// @Accept  json
// @Produce  json
// @Param   "请求相关参数：无"
// @success 200 {object} api.TempVoucherResp{} "返回JSON数据"
// @Router /video/sts_voucher [post]
func GetTempVoucher(c *gin.Context) {
	xc := xng.NewXContext(c)
	resp, err := videoService.TempVoucher()
	if err != nil || resp == nil {
		xlog.ErrorC(xc, "failed to get sts temp voucher, err:%v", err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.ReplyOK(resp)
}

// HandleTransCompleted godoc
// @Summary 转码状态处理接口
// @Description 客户端完成直传，点击发表后调用此接口，根据转码状态进行处理并将结果加入通知列表
// @Accept  json
// @Produce  json
// @Param body body api.CheckStatusReq true "请求相关的参数"
// @success 200 {object} struct{} "返回数据：无"
// @Router /video/handle_trans_completed [post]
func HandleTransCompleted(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req api.CheckStatusReq
	if !xc.GetReqObject(&req) {
		return
	}
	if req.Qid <= 0 || req.Aid <= 0 || req.ResId <= 0 {
		xc.ReplyFail(lib.CodePara)
		return
	}
	err := videoService.HandleTransCompleted(&req)
	if err != nil {
		xlog.ErrorC(xc, "failed to handle trans resource, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.ReplyOK(struct{}{})
}

// AppUploadCallback godoc
// @Summary 客户端回调
// @Description 客户端文件上传成功后回调该接口；该接口不对外开放，需要由业务接口调用
// @Accept  json
// @Produce  json
// @Param body body api.AppUploadCallbackReq true "请求相关的参数"
// @success 200 {object} api.MediaInfoResp{} "返回JSON数据"
// @Router /video/app_upload_callback [post]
func AppUploadCallback(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req api.AppUploadCallbackReq
	if !xc.GetReqObject(&req) {
		return
	}
	if req.QeTag == "" || req.Qid <= 0 {
		xc.ReplyFail(lib.CodePara)
		return
	}
	ty := req.FileType
	statusParam := api.SetUploadStatusReq{
		QeTag:  req.QeTag,
		Qid:    req.Qid,
		Status: 2,
	}
	err := videoService.SetUploadStatus(&statusParam, ty)
	if err != nil {
		xlog.ErrorC(xc, "failed to set upload status, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	mediaParam := api.MediaInfoReq{
		Mid:   req.Mid,
		Qid:   req.Qid,
		QeTag: req.QeTag,
	}
	resp, err := videoService.MediaInfo(&mediaParam, ty)
	if err != nil || resp == nil {
		xlog.ErrorC(xc, "failed to get video info, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = videoService.AppUploadCallback(req.Qid)
	if err != nil {
		xlog.ErrorC(xc, "failed to copy resource bucket, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	//app不使用大文件的转码环节,trans字段默认为转码完成（1）
	qry := bson.M{"_id": req.Qid}
	mt := time.Now().UnixNano() / 1e6
	updata := bson.M{"$set": bson.M{"trans": resource.ResStatusTrans, "mt": mt}}
	err = DaoByQid.UpdateResourceDoc(req.Qid, qry, updata)
	if err != nil {
		xlog.Error("update resource doc error:%v, qid:%v", req.Qid, err)
		return
	}
	xc.ReplyOK(resp)
}

// GetResourceInfoByEtag godoc
// @Summary 根据etag查询资源信息
// @Description 根据etag查询资源信息
// @Accept  json
// @Produce  json
// @Param body body api.UploadStatusReq true "请求相关的参数"
// @success 200 {object} api.XngResourceInfoDoc{} "返回JSON数据"
// @Router /video/app_upload_callback [post]
func GetResourceInfoByEtag(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req api.UploadStatusReq
	if !xc.GetReqObject(&req) {
		return
	}
	if req.QeTag == "" {
		xc.ReplyFail(lib.CodePara)
		return
	}
	resourceInfo, err := videoService.GetResDocByQeTag(req.QeTag)
	if err != nil {
		xlog.ErrorC(xc, "failed to GetResDocByQeTag, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.ReplyOK(resourceInfo)
}
