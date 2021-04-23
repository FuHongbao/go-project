package trans

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/resource"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/trans"
	urlService "xgit.xiaoniangao.cn/xngo/service/resources_center/service/url"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils"
)

func vaildMsgParam(ctx context.Context, msgData string) (userData trans.AlbumTransCallBack, err error) {
	data, err := base64.StdEncoding.DecodeString(msgData)
	if err != nil {
		xlog.ErrorC(ctx, "fail to decode base64, data:%s, err:%v", msgData, err)
		return
	}
	err = json.Unmarshal(data, &userData)
	if err != nil {
		return
	}
	return
}

type MqResInfo struct {
	JobId       string  `json:"jobID"`
	ResId       string  `json:"id"`
	Type        int     `json:"ty"`
	Size        int64   `json:"size"`
	QeTag       string  `json:"qetag"`
	Upt         int64   `json:"upt"`
	Fmt         string  `json:"fmt"`
	W           int     `json:"w"`
	H           int     `json:"h"`
	Du          float64 `json:"du,omitempty"`
	Cover       string  `json:"cover,omitempty"`
	Code        string  `json:"code,omitempty"`
	Ort         int     `json:"ort,omitempty"`
	Url         string  `json:"url"`
	UrlInternal string  `json:"url_internal"`
	UserData    string  `json:"user_data"`
}
type MqTransVideoMsg struct {
	Status int       `json:"status"`
	Data   MqResInfo `json:"data"`
}

func CallbackForTrans(c *gin.Context) {
	xc := xng.NewXContext(c)
	message := &api.MNSMessageData{}
	if !xc.GetReqObject(&message) {
		return
	}
	if message.UserData == "" {
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	param, err := utils.GetMNSCallBackData(xc, message.UserData)
	if err != nil {
		xlog.ErrorC(xc, "CallbackForTrans.GetMNSCallBackData failed, req:%v, err:%v", message, err)
		xc.ReplyFail(lib.CodePara)
		return
	}
	if param.JobType != api.MTSJobTypeTransVideo {
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	var backData api.MultiVideoUserData
	err = json.Unmarshal([]byte(param.UserData), &backData)
	if err != nil {
		xlog.ErrorC(xc, "CallbackForTrans.GetMNSCallBackData failed, req:%v, err:%v", message, err)
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	if message.State != api.NotifyStatusSuccess { //转码失败
		data := api.ResDocWithCoverUrl{ResId: param.Key, UserData: backData.UserData, OldId: backData.OldId, JobId: message.JobID}
		err = resource.NotifyVideoTransMq(xc, backData.UserService, &data, 0)
		if err != nil {
			xlog.ErrorC(xc, "CallbackForTrans.NotifyVideoTransMq failed, req:%v, err:%v", message, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	var qdoc *api.XngResourceInfoDoc //防止回调失败重试，数据库重复插入
	if param.Kind != api.ResourceTypeGuideVideo {
		qdoc, err = resource.ByID(xc, param.Key)
		if err != nil {
			xlog.ErrorC(xc, "CallbackForTrans.ByID, req:%v, err:%v", message, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
	}
	if qdoc != nil {
		xlog.DebugC(xc, "CallbackForTrans.doc already exists:[%v]", qdoc.ResId)
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	qdoc, err = resource.HandleXngResourcesInfo(xc, param.Key, param.Project, param.Kind, backData.Qetag, param.Path)
	if err != nil {
		data := api.ResDocWithCoverUrl{ResId: param.Key, UserData: backData.UserData, OldId: backData.OldId, JobId: message.JobID}
		err = resource.NotifyVideoTransMq(xc, backData.UserService, &data, 0)
		if err != nil {
			xlog.ErrorC(xc, "CallbackForTrans.NotifyVideoTransMq failed, req:%v, err:%v", message, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		xlog.ErrorC(xc, "CallbackForTrans.HandleXngResourcesInfo failed, err:[%v], req:[%v]", err, param)
		xc.Reply(http.StatusNoContent, nil)
		return
	}
	//更新源资源的转码信息
	if backData.OldId != "" && param.Kind != api.ResourceTypeGuideVideo {
		err = resource.UpdateResTransRecord(xc, backData.OldId, param.Key, backData.TplId)
		if err != nil {
			data := api.ResDocWithCoverUrl{ResId: param.Key, UserData: backData.UserData, OldId: backData.OldId, JobId: message.JobID}
			err = resource.NotifyVideoTransMq(xc, backData.UserService, &data, 0)
			if err != nil {
				xlog.ErrorC(xc, "CallbackForTrans.NotifyVideoTransMq failed, req:%v, err:%v", message, err)
				xc.ReplyFail(lib.CodeSrv)
				return
			}
			xlog.ErrorC(xc, "CallbackForTrans.UpdateResTransRecord failed, err:[%v], req:[%v]", err, param)
			xc.Reply(http.StatusNoContent, nil)
			return
		}
		xlog.DebugC(xc, "CallbackForTrans add trans record oldID:[%s], transID:[%s] success", backData.OldId, param.Key)
	}
	//通过mq推送转码资源信息
	var url, urlInternal string
	if qdoc.Cover != 0 {
		cover := fmt.Sprintf("%d", qdoc.Cover)
		url, urlInternal = urlService.GetImageURL(xc, cover, "imageMogr2/thumbnail/750x500/format/jpg")
	}
	resDoc := api.ResDocWithCoverUrl{
		ResId:            strconv.FormatInt(qdoc.ResId, 10),
		Type:             qdoc.Type,
		Size:             qdoc.Size,
		QeTag:            qdoc.QeTag,
		Upt:              qdoc.Upt,
		Fmt:              qdoc.Fmt,
		W:                qdoc.W,
		H:                qdoc.H,
		Du:               qdoc.Du,
		Cover:            fmt.Sprintf("%d", qdoc.Cover),
		Code:             qdoc.Code,
		Ort:              qdoc.Ort,
		CoverUrl:         url,
		CoverUrlInternal: urlInternal,
		JobId:            message.JobID,
		OldId:            backData.OldId,
		UserData:         backData.UserData,
	}
	err = resource.NotifyVideoTransMq(xc, backData.UserService, &resDoc, 1)
	if err != nil {
		xlog.ErrorC(xc, "CallbackForTrans.NotifyMediaInfoMq failed, req:%v, err:%v", message, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.Reply(http.StatusNoContent, nil)
	return
}
