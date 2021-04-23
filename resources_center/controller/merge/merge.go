package merge

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByQid"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/callbackMQ"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/merge"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/resource"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/sts"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils"
)

type ReqVideoMerge struct {
	Ids      []string `json:"ids"`
	Kind     int      `json:"type"`
	Product  int      `json:"prod"`
	Project  string   `json:"proj"`
	W        int      `json:"w"`
	H        int      `json:"h"`
	Path     string   `json:"path"`
	UserData string   `json:"user_data"`
}
type RespVideoMerge struct {
	JobID  string `json:"jobID"`
	Status int    `json:"status"`
}

type ResInfo struct {
	JobId    string  `json:"jobID"`
	ResId    string  `json:"id"`
	Type     int     `json:"ty"`
	Size     int64   `json:"size"`
	QeTag    string  `json:"qetag"`
	Upt      int64   `json:"upt"`
	Fmt      string  `json:"fmt"`
	W        int     `json:"w"`
	H        int     `json:"h"`
	Du       float64 `json:"du,omitempty"`
	Cover    string  `json:"cover,omitempty"`
	Code     string  `json:"code,omitempty"`
	Ort      int     `json:"ort,omitempty"`
	Path     string  `json:"path,omitempty"`
	UserData string  `json:"user_data"`
}

type MqMergeVideoMsg struct {
	Status int     `json:"status"`
	Data   ResInfo `json:"data"`
}

func VideoMerge(c *gin.Context) {
	xc := xng.NewXContext(c)
	req := &ReqVideoMerge{}
	if !xc.GetReqObject(&req) {
		return
	}
	if len(req.Ids) < 2 || req.H <= 0 || req.W <= 0 || req.Kind <= 0 || req.Product <= 0 || req.Project == "" {
		xc.ReplyFail(lib.CodePara)
		return
	}
	conf.MergeCounter.Inc()
	if req.Path != "" && !strings.HasSuffix(req.Path, "/") {
		req.Path += "/"
	}
	ch := make(chan int64, 1)
	go resource.GetDistributedId(ch)
	stsName := resource.GetMtsStsName(req.Kind)
	stsData, err := sts.GetUpToken(xc, stsName)
	if err != nil {
		xlog.ErrorC(xc, "failed to GetUpToken, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if stsData == nil {
		xlog.ErrorC(xc, "failed to GetUpToken, sts data is nil")
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	qid := <-ch
	if qid <= 0 {
		xlog.ErrorC(xc, "failed to GetDistributedId, id is nil")
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	var jobId string
	var confURL string
	var status int
	key := strconv.FormatInt(qid, 10)
	if len(req.Ids) < 4 { //拼接视频数五个以内直接进行拼接作业
		jobId, status, err = merge.HandleVideoMerge(xc, req.Ids, req.W, req.H, stsData, key, req.Path, req.Kind, req.Product, req.Project, req.UserData)
		if err != nil {
			xlog.ErrorC(xc, "VideoMerge.HandleVideoMerge failed, req:%v, err:%v", req, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
	} else { //五个及以上需要先上传配置文件
		confURL, err = merge.UploadMergeVideoConf(xc, stsData, req.Ids, stsData.Bucket, key, conf.Env)
		if err != nil {
			xlog.ErrorC(xc, "VideoMerge.UploadMergeVideoConf failed, req:%v, err:%v", req, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		jobId, status, err = merge.HandleLargeCntVideoMerge(xc, confURL, req.Ids[0], req.W, req.H, stsData, key, req.Path, req.Kind, req.Product, req.Project, req.UserData)
		if err != nil {
			xlog.ErrorC(xc, "VideoMerge.HandleVideoMerge failed, req:%v, err:%v", req, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
	}

	resp := RespVideoMerge{Status: status, JobID: jobId}
	xc.ReplyOK(resp)
}

func vaildMsgParam(ctx context.Context, msgData string) (userData merge.ResMergeCallBack, err error) {
	dataByte, err := base64.StdEncoding.DecodeString(msgData)
	if err != nil {
		xlog.ErrorC(ctx, "fail to decode base64, data:%s, err:%v", msgData, err)
		return
	}
	err = json.Unmarshal(dataByte, &userData)
	if err != nil {
		return
	}
	return
}

func ResultCallBack(c *gin.Context) {
	xc := xng.NewXContext(c)
	message := &api.MNSMessageData{}
	if !xc.GetReqObject(&message) {
		return
	}
	if message.UserData == "" {
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	//param, err := vaildMsgParam(xc, message.UserData)
	param, err := utils.GetMNSCallBackData(xc, message.UserData)
	if err != nil {
		xlog.ErrorC(xc, "ResultCallBack.GetMNSCallBackData failed, req:%v, err:%v", message, err)
		xc.ReplyFail(lib.CodePara)
		return
	}
	if param.JobType != api.MTSJobTypeMergeVideo {
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	topic, ok := api.MqMergeTopicMap[param.Product]
	if !ok {
		xlog.ErrorC(xc, "unknow type of product:%d", param.Product)
		return
	}
	tagStr, ok := api.MqMergeTagMap[param.Kind]
	if !ok {
		xlog.ErrorC(xc, "unknow type of resource:%d", param.Kind)
		return
	}
	tag := fmt.Sprintf("src_%s:%s", param.Project, tagStr)
	if message.State != api.NotifyStatusSuccess { //合并失败
		data := ResInfo{JobId: message.JobID, UserData: param.UserData}
		mqMsg := MqMergeVideoMsg{Status: 0, Data: data}
		err = callbackMQ.NotifyByMq(xc, topic, tag, mqMsg)
		if err != nil {
			xlog.ErrorC(xc, "ResultCallBack.NotifyByMq, req:%v, err:%v", message, err)
			return
		}
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	/*
		stsData, err := sts.GetUpToken(xc, stsName)
		if err != nil {
			xlog.ErrorC(xc, "failed to GetUpToken, req:%v, err:%v", message, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		if stsData == nil {
			xlog.ErrorC(xc, "failed to GetUpToken, sts data is nil")
			xc.ReplyFail(lib.CodeSrv)
			return
		}
	*/
	var qdoc *api.XngResourceInfoDoc //防止回调失败重试，数据库重复插入
	qdoc, err = resource.ByID(xc, param.Key)
	if err != nil {
		xlog.ErrorC(xc, "ResultCallBack.ByID, req:%v, err:%v", message, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if qdoc == nil {
		stsName := resource.GetMtsStsName(param.Kind)
		mtsInfo, err := resource.GetAliMtsClient(xc, stsName)
		if err != nil {
			xlog.ErrorC(xc, "ResultCallBack.GetAliMtsClient failed, req:%v, err:%v", message, err)
			xc.ReplyFail(lib.CodeSrv)
		}
		qdoc, err = resource.GetOssCallbackVideoDoc(xc, mtsInfo.Client, mtsInfo.Sts, param.Key, param.Path, "", param.Project, mtsInfo.Sts.Bucket, param.Kind)
		//qdoc, err = resource.OrganizeVideoDoc(xc, param.Key, param.Path, "", param.Project, stsData, param.Kind)
		if err != nil {
			xlog.ErrorC(xc, "ResultCallBack.OrganizeVideoDoc, req:%v, err:%v", message, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		err = DaoByQid.InsertResourceDoc(qdoc.ResId, qdoc)
		if err != nil {
			xlog.ErrorC(xc, "ResultCallBack.InsertResourceDoc, req:%v, err:%v", message, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
	}
	resDoc := ResInfo{
		JobId:    message.JobID,
		ResId:    strconv.FormatInt(qdoc.ResId, 10),
		Type:     qdoc.Type,
		Size:     qdoc.Size,
		QeTag:    "",
		Upt:      qdoc.Upt,
		Fmt:      qdoc.Fmt,
		W:        qdoc.W,
		H:        qdoc.H,
		Du:       qdoc.Du,
		Cover:    strconv.FormatInt(qdoc.Cover, 10),
		Code:     qdoc.Code,
		Ort:      qdoc.Ort,
		Path:     param.Path,
		UserData: param.UserData,
	}
	mqMsg := MqMergeVideoMsg{Status: 1, Data: resDoc}
	err = callbackMQ.NotifyByMq(xc, topic, tag, mqMsg)
	if err != nil {
		xlog.ErrorC(xc, "ResultCallBack.NotifyByMq failed, req:%v, err:%v", message, err)
		return
	}
	xc.Reply(http.StatusNoContent, nil)
}
