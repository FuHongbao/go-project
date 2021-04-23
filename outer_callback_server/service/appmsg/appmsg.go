package appmsg

import (
	"xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo/bson"
	"xgit.xiaoniangao.cn/xngo/service/user_message_center_api"
)

type AppRawMsg struct {
	Mid        int64  `json:"mid"`
	Channel    int    `json:"channel"`
	AppVersion string `json:"app_version"`
	MsgType    string `json:"type"`
	Content    string `json:"content,omitempty"`
	CreateTime int64  `json:"ct"`
}

//NormalizeAppRawMsg 组装消息
func NormalizeAppRawMsg(appRawMsg *AppRawMsg) (*user_message_center_api.RevReq, bool) {
	var (
		xngMsg = &user_message_center_api.RevReq{}
	)
	xngMsg.Mid = appRawMsg.Mid
	xngMsg.MsgId = bson.NewObjectId().Hex()
	xngMsg.Channel = appRawMsg.Channel
	xngMsg.Ct = int64(appRawMsg.CreateTime) * 1000
	xngMsg.AppVersion = appRawMsg.AppVersion
	switch appRawMsg.MsgType {
	case "text":
		xngMsg.Type = user_message_center_api.MsgTypeText
		xngMsg.Body = map[string]interface{}{"txt": appRawMsg.Content}
	default:
		return nil, false
	}

	return xngMsg, true
}
