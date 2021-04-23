package resource

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo/bson"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByQid"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/resource"
	urlService "xgit.xiaoniangao.cn/xngo/service/resources_center/service/url"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils/alioss"
)

// GetUploadStatus godoc
// @Summary 获取资源上传状态接口
// @Description 客户端调用此接口获取资源上传状态（已上传：2；未上传：0）
// @Accept  json
// @Produce  json
// @Param   body body api.UploadStatusReq true "请求相关的参数"
// @success 200 {object} api.UploadStatusResp "返回JSON数据"
// @Router /resource/get_upload_status [post]
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
	req.QeTag = utils.GetQetagByResType(req.Type, req.QeTag)
	status, _, err := resource.GetUploadStatus(xc, req.QeTag)
	if err != nil {
		xlog.ErrorC(xc, "failed to get upload status, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.ReplyOK(&api.UploadStatusResp{Status: status})
}

type DealOssCallBackReq struct {
	alioss.OssCallBack
	MyVar string `json:"my_var" binding:"required"`
}

type DealOssCallBackResp struct {
	ID string `json:"id"`
}

// DealOssCallBack
// @Summary 资源中心上传回调接口
// @Description 资源上传完成后阿里云回调该接口；处理上传结果并通过mq通知业务端
// @Accept  json
// @Produce  json
// @Param body api.DealOssCallBackReq true "请求相关的参数"
// @success 200 {object} api.DealOssCallBackResp{} "返回JSON数据"
// @Router /resource/oss_callback [post]
func DealOssCallBack(c *gin.Context) {
	xc := xng.NewXContext(c)
	req := &DealOssCallBackReq{}
	if !xc.GetReqObject(&req) {
		return
	}
	start := time.Now()
	customVar, err := validateReq(xc, req)
	if err != nil {
		xlog.ErrorC(xc, "request params is nil, req:%v, err:%v", req, err)
		xc.ReplyFailWithDetail(lib.CodePara, err.Error())
		return
	}
	resKey := req.Filename
	if customVar.Kind == api.ResourceTypeLive {
		ind := strings.LastIndex(resKey, "/")
		resKey = req.Filename[ind+1:]
	}
	//防止前端一套配置多次上传，产生多次回调
	doc, err := resource.ByID(xc, req.Filename)
	if doc != nil && err == nil {
		resp := DealOssCallBackResp{
			ID: fmt.Sprintf("%d", doc.ResId),
		}
		xc.ReplyOK(resp)
	}
	customVar.QeTag = utils.GetQetagByResType(customVar.Kind, customVar.QeTag)
	ret, err := resource.DealOssCallBackMsg(xc, req.OssCallBack, customVar)
	if err != nil {
		xlog.ErrorC(xc, "failed to deal oss callback message, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = resource.SetUploadLocalCache(xc, resKey, ret) //添加5分钟缓存
	if err != nil {
		xlog.ErrorC(xc, "DealOssCallBack failed to Set Upload LocalCache, req:%v, err:%v", req, err)
	}
	if customVar.NoMq == 0 { //判断是否关闭mq回调
		// mq 回调
		err = resource.NotifyUploadMsg(xc, customVar.Product, customVar.Project, ret, req.MyVar)
		if err != nil {
			xlog.ErrorC(xc, "NotifyUploadMsg failed, req:%v, err:%v", req, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
	}
	xlog.DebugC(xc, "DealOssCallBack deal file:[%s] use time:[%d]", req.Filename, time.Since(start))
	resp := DealOssCallBackResp{
		ID: fmt.Sprintf("%d", ret.ResId),
	}
	xc.ReplyOK(resp)
}

func validateReq(ctx context.Context, req *DealOssCallBackReq) (customVar *api.CallbackCustomParam, err error) {
	v := req.MyVar
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		xlog.ErrorC(ctx, "fail to decode base64, v:%s, err:%v", v, err)
		return
	}

	xlog.DebugC(ctx, "data:%s", string(data))

	customVar = &api.CallbackCustomParam{}
	if err = json.Unmarshal(data, &customVar); err != nil {
		xlog.ErrorC(ctx, "fail to decode json, data:%v, err:%v", data, err)
		return
	}

	if customVar.QeTag == "" || customVar.Project == "" || customVar.Product == 0 || req.Filename == "" || customVar.Kind == 0 {
		//如果缺少prod，proj，qid无法合理发送通知给业务端，这里感觉由业务端处理比较好（前端接收到阿里云的上传失败，然后再准确通知业务端）
		err = fmt.Errorf("callback param err")
		xlog.ErrorC(ctx, "request params is nil, customVar:%v, err:%v", customVar, err)
		return
	}
	return
}

//type MqResInfo struct {
//	ResId    string  `json:"id"`
//	Type     int     `json:"ty"`
//	Size     int64   `json:"size"`
//	QeTag    string  `json:"qetag"`
//	Upt      int64   `json:"upt"`
//	Fmt      string  `json:"fmt"`
//	W        int     `json:"w"`
//	H        int     `json:"h"`
//	Du       float64 `json:"du,omitempty"`
//	Cover    string  `json:"cover,omitempty"`
//	Code     string  `json:"code,omitempty"`
//	Ort      int     `json:"ort,omitempty"`
//	UserData string  `json:"user_data"`
//}
//type MqVideoInfoMsg struct {
//	Status int       `json:"status"`
//	Data   MqResInfo `json:"data"`
//}

func DealMediaInfoCallBack(c *gin.Context) {
	xc := xng.NewXContext(c)
	message := &api.MNSMessageData{}
	if !xc.GetReqObject(&message) {
		return
	}
	if message.UserData == "" {
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	//解析阿里云回调中的自定义信息
	param, err := utils.GetMNSCallBackData(xc, message.UserData)
	if err != nil {
		xlog.ErrorC(xc, "DealMediaInfoCallBack.GetMNSCallBackData failed, req:%v, err:%v", message, err)
		xc.ReplyFail(lib.CodePara)
		return
	}
	if param.JobType != api.MTSJobTypeVideoInfo && param.JobType != api.MTSJobTypeOpVideoInfo {
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	var backData api.MultiVideoUserData
	err = json.Unmarshal([]byte(param.UserData), &backData)
	if err != nil {
		xlog.ErrorC(xc, "DealMediaInfoCallBack.GetMNSCallBackData failed, req:%v, err:%v", message, err)
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	xlog.InfoC(xc, "DealMediaInfoCallBack recv callback data:[%v], [%v]", param, backData)
	if message.State != api.NotifyStatusSuccess { //阿里云处理视频信息失败（视频存在问题等原因），通知失败
		data := api.ResDocWithCoverUrl{ResId: param.Key, QeTag: backData.Qetag, UserData: backData.UserData}
		err = resource.NotifyMediaInfoMq(xc, param.Product, param.Project, backData.UserService, &data, 0)
		if err != nil {
			xlog.ErrorC(xc, "DealMediaInfoCallBack.NotifyMediaInfoMq failed, req:%v, err:%v", message, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		//mqMsg := MqVideoInfoMsg{Status: 0, Data: data}
		//err = callbackMQ.NotifyByMq(xc, topic, tag, mqMsg)
		//if err != nil {
		//	xlog.ErrorC(xc, "DealMediaInfoCallBack.NotifyByMq, req:%v, err:%v", message, err)
		//	return
		//}
		xlog.InfoC(xc, "DealMediaInfoCallBack mediaInfo task failed data:[%v], [%v]", param, backData)
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	//媒体信息作业成功则查询jobid对应的结果
	stsName := resource.GetMtsStsName(param.Kind)
	mtsInfo, err := resource.GetAliMtsClient(xc, stsName)
	if err != nil {
		xlog.ErrorC(xc, "DealMediaInfoCallBack.GetAliMtsClient failed, req:%v, err:%v", param, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	doc, err := resource.QueryMediaInfoAsync(xc, mtsInfo.Client, param.Key, param.Kind, message.JobID, param.Project)
	if err != nil {
		xlog.ErrorC(xc, "DealMediaInfoCallBack.QueryMediaInfoAsync failed, req:%v, err:%v", param, err)
		data := api.ResDocWithCoverUrl{ResId: param.Key, QeTag: backData.Qetag, UserData: backData.UserData}
		err = resource.NotifyMediaInfoMq(xc, param.Product, param.Project, backData.UserService, &data, 0)
		if err != nil {
			xlog.ErrorC(xc, "DealMediaInfoCallBack.NotifyMediaInfoMq failed, req:%v, err:%v", message, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	doc.QeTag = backData.Qetag
	//尝试查询库内资源记录
	qDoc, err := resource.ByID(xc, param.Key)
	if err != nil {
		xlog.ErrorC(xc, "DealMediaInfoCallBack.ByID failed, req:%v, err:%v", param, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if qDoc != nil && qDoc.Fmt != "" && qDoc.Code != "" { //阿里云回调重复过滤
		xlog.InfoC(xc, "DealMediaInfoCallBack mediaInfo task already exists, data:[%v], [%v]", param, backData)
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	switch param.JobType {
	case api.MTSJobTypeVideoInfo:
		if qDoc == nil {
			xlog.InfoC(xc, "DealMediaInfoCallBack.ByID not found, key:[%v]", param.Key)
			xc.Reply(http.StatusNoContent, nil)
			return
		}
		doc.Cover = qDoc.Cover
		doc.CoverTp = qDoc.CoverTp
		qry := bson.M{"_id": doc.ResId}
		updata := bson.M{"$set": bson.M{"code": doc.Code, "fmt": doc.Fmt, "size": doc.Size, "du": doc.Du}}
		err = DaoByQid.UpdateResourceDoc(doc.ResId, qry, updata)
		if err != nil {
			xlog.ErrorC(xc, "DealMediaInfoCallBack.UpdateResourceDoc failed, err:[%v], update:[%v]", err, doc)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
	case api.MTSJobTypeOpVideoInfo:
		snapId, err := resource.GetVideoSnapShot(xc, param.Key, doc.Du, doc.Code, param.Kind, param.Project)
		if err != nil {
			data := api.ResDocWithCoverUrl{ResId: param.Key, QeTag: backData.Qetag, UserData: backData.UserData}
			err = resource.NotifyMediaInfoMq(xc, param.Product, param.Project, backData.UserService, &data, 0)
			if err != nil {
				xlog.ErrorC(xc, "DealMediaInfoCallBack.NotifyMediaInfoMq failed, req:%v, err:%v", message, err)
				xc.ReplyFail(lib.CodeSrv)
				return
			}
			xlog.ErrorC(xc, "DealMediaInfoCallBack.GetVideoSnapShot failed, err:[%v], req:[%v]", err, param)
			xc.Reply(http.StatusNoContent, nil)
			return
		}
		doc.Cover = snapId
		doc.CoverTp = resource.VideoCoverType
		err = resource.UpInsertMediaInfoToDB(doc) //存库失败：使用阿里云得回调重试再次执行流程
		if err != nil {
			xlog.ErrorC(xc, "DealMediaInfoCallBack.AddNewMediaInfoToDB failed, req:%v, err:%v", message, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
	default:
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	//if param.JobType != api.MTSJobTypeMultiVideoInfo { //分片上传时库内会有数据记录，此时只更新fmt和code字段
	//	doc.Cover = qDoc.Cover
	//	doc.Code = qDoc.Code
	//	doc.CoverTp = qDoc.CoverTp
	//	resId, err := strconv.ParseInt(param.Key, 10, 64)
	//	if err != nil {
	//		data := api.ResDocWithCoverUrl{ResId: param.Key, UserData: backData.UserData}
	//		err = resource.NotifyMediaInfoMq(xc, param.Product, param.Project, backData.UserService, &data, 0)
	//		if err != nil {
	//			xlog.ErrorC(xc, "DealMediaInfoCallBack.NotifyMediaInfoMq failed, req:%v, err:%v", message, err)
	//			xc.ReplyFail(lib.CodeSrv)
	//			return
	//		}
	//		xlog.ErrorC(xc, "DealMediaInfoCallBack.ParseInt failed, err:[%v], req:[%v]", err, param)
	//		xc.Reply(http.StatusNoContent, nil)
	//		return
	//	}
	//	err = DaoByQid.UpInsertResourceDoc(resId, doc)
	//	if err != nil {
	//		xlog.ErrorC(xc, "DealMediaInfoCallBack.UpdateResourceDoc failed, err:[%v], update:[%v]", err, doc)
	//		xc.ReplyFail(lib.CodeSrv)
	//		return
	//	}
	//} else { //非分片上传不会存在记录，需要新生成截图和信息存库
	//	snapId, err := resource.GetVideoSnapShot(xc, param.Key, doc.Du, doc.Code, param.Kind, param.Project)
	//	if err != nil {
	//		data := api.ResDocWithCoverUrl{ResId: param.Key, UserData: backData.UserData}
	//		err = resource.NotifyMediaInfoMq(xc, param.Product, param.Project, backData.UserService, &data, 0)
	//		if err != nil {
	//			xlog.ErrorC(xc, "DealMediaInfoCallBack.NotifyMediaInfoMq failed, req:%v, err:%v", message, err)
	//			xc.ReplyFail(lib.CodeSrv)
	//			return
	//		}
	//		xlog.ErrorC(xc, "DealMediaInfoCallBack.GetVideoSnapShot failed, err:[%v], req:[%v]", err, param)
	//		xc.Reply(http.StatusNoContent, nil)
	//		return
	//	}
	//	doc.Cover = snapId
	//	doc.CoverTp = resource.VideoCoverType
	//	err = resource.AddNewMediaInfoToDB(doc) //存库失败：使用阿里云得回调重试再次执行流程
	//	if err != nil {
	//		xlog.ErrorC(xc, "DealMediaInfoCallBack.AddNewMediaInfoToDB failed, req:%v, err:%v", message, err)
	//		xc.ReplyFail(lib.CodeSrv)
	//		return
	//	}
	//}
	_ = resource.AddMediaInfoCache(xc, doc, param.Key)
	//资源信息通过mq推送给业务端
	cover := fmt.Sprintf("%d", doc.Cover)
	url, urlInternal := urlService.GetImageURL(xc, cover, "imageMogr2/thumbnail/750x500/format/jpg")
	resDoc := api.ResDocWithCoverUrl{
		ResId:            fmt.Sprintf("%d", doc.ResId),
		Type:             doc.Type,
		Size:             doc.Size,
		QeTag:            doc.QeTag,
		Upt:              doc.Upt,
		Fmt:              doc.Fmt,
		W:                doc.W,
		H:                doc.H,
		Du:               doc.Du,
		Cover:            fmt.Sprintf("%d", doc.Cover),
		Code:             doc.Code,
		Ort:              doc.Ort,
		CoverUrl:         url,
		CoverUrlInternal: urlInternal,
		UserData:         backData.UserData,
	}
	err = resource.NotifyMediaInfoMq(xc, param.Product, param.Project, backData.UserService, &resDoc, 1)
	if err != nil {
		xlog.ErrorC(xc, "DealMediaInfoCallBack.NotifyMediaInfoMq failed, req:%v, err:%v", message, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.Reply(http.StatusNoContent, nil)
	return
	//data := MqResInfo{
	//	ResId:    param.Key,
	//	Type:     doc.Type,
	//	Size:     doc.Size,
	//	QeTag:    doc.QeTag,
	//	Upt:      doc.Upt,
	//	Fmt:      doc.Fmt,
	//	W:        doc.W,
	//	H:        doc.H,
	//	Du:       doc.Du,
	//	Ort:      doc.Ort,
	//	Cover:    fmt.Sprintf("%d", qDoc.Cover),
	//	Code:     doc.Code,
	//	UserData: backData.UserData,
	//}
	//mqMsg := MqVideoInfoMsg{Status: 1, Data: data}
	//err = callbackMQ.NotifyByMq(xc, topic, tag, mqMsg)
	//if err != nil {
	//	xlog.ErrorC(xc, "DealMediaInfoCallBack.NotifyByMq, req:%v, err:%v", message, err)
	//	return
	//}
}
