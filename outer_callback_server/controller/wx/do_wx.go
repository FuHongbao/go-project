package wx

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"net/http"
	"strconv"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/conf"

	proxy "xgit.xiaoniangao.cn/xngo/service/outer_callback_server/mq"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/service/phpwxreceive"
	userService "xgit.xiaoniangao.cn/xngo/service/outer_callback_server/service/user"
	wxmsgService "xgit.xiaoniangao.cn/xngo/service/outer_callback_server/service/wxmsg"

	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/utils"
	"xgit.xiaoniangao.cn/xngo/service/user_message_center_api"
)

func inGray(msgType string, openid string, channel int) bool {
	if channel == user_message_center_api.ChannelXNGService {
		msgTypeInGray := utils.InStrArray(conf.C.MpGrayMsgTypes, "all") || utils.InStrArray(conf.C.MpGrayMsgTypes, msgType)
		openidInGray := utils.InStrArray(conf.C.MpGrayOpenids, "all") || utils.InStrArray(conf.C.MpGrayOpenids, openid)
		//return (msgTypeInGray && openidInGray) || msgType == "event"
		return msgTypeInGray && openidInGray
	} else if channel == user_message_center_api.ChannelXNGMiniApp {
		msgTypeInGray := utils.InStrArray(conf.C.MaGrayMsgTypes, "all") || utils.InStrArray(conf.C.MaGrayMsgTypes, msgType)
		openidInGray := utils.InStrArray(conf.C.MaGrayOpenids, "all") || utils.InStrArray(conf.C.MaGrayOpenids, openid)
		return msgTypeInGray && openidInGray
	} else if channel == user_message_center_api.ChannelXNGSubscribe {
		msgTypeInGray := utils.InStrArray(conf.C.SubGrayMsgTypes, "all") || utils.InStrArray(conf.C.SubGrayMsgTypes, msgType)
		openidInGray := utils.InStrArray(conf.C.SubGrayOpenids, "all") || utils.InStrArray(conf.C.SubGrayOpenids, openid)
		//return (msgTypeInGray && openidInGray) || msgType == "event"
		return msgTypeInGray && openidInGray
	} else if channel == user_message_center_api.ChannelGameIdiomMiniApp {
		return true
	}
	return true
}

func getToken(channel int) string {
	switch channel {
	case user_message_center_api.ChannelXNGService:
		return conf.C.WxToken["xngservice"]
	case user_message_center_api.ChannelXNGMiniApp:
		return conf.C.WxToken["xngminiapp"]
	case user_message_center_api.ChannelXNGSubscribe:
		return conf.C.WxToken["xngsubscribe"]
	case user_message_center_api.ChannelGameIdiomMiniApp:
		return conf.C.WxToken["gameidiomminiapp"]
	case user_message_center_api.ChannelGameDuetMiniApp:
		return conf.C.WxToken["gameduetminiapp"]
	}

	xlog.Error("token is empty")
	return ""
}

func doValidate(c *gin.Context, channel int) {
	var (
		err     error
		wxquery = &wxmsgService.WxQuery{}
		xc      = xng.NewXContext(c)
		token   = getToken(channel)
	)
	if err = xc.BindQuery(wxquery); err != nil {
		xlog.Error("bind query fail, err=%s", err)
		xc.String(http.StatusOK, "")
		return
	}

	xlog.Debug("wxquery=%v", *wxquery)

	if wxmsgService.CheckWxSignature(wxquery, token) {
		c.String(http.StatusOK, wxquery.Echostr)
	} else {
		xlog.Error("check signature fail")
		c.String(http.StatusOK, "")
	}
}

func doWxMsg(c *gin.Context, channel int) {
	var (
		err     error
		req     = &wxmsgService.WxRawMsg{}
		xc      = xng.NewXContext(c)
		wxquery = &wxmsgService.WxQuery{}
		token   = getToken(channel)
	)
	if err = xc.BindQuery(wxquery); err != nil {
		xlog.Error("bind query fail, err=%s", err)
		xc.String(http.StatusOK, "")
		return
	}
	//小程序支持json和xml两种类型，所以这里判断一下，小程序的暂时必须配成json，因为过渡阶段转发到php，php小程序只支持json
	if xc.ContentType() == gin.MIMEJSON {
		err = xc.ShouldBindBodyWith(req, binding.JSON)
	} else {
		err = xc.ShouldBindBodyWith(req, binding.XML)
	}
	if err != nil {
		msgData, ok := xc.Get(gin.BodyBytesKey)
		if !ok {
			xlog.Error("get body fail")
		}
		msgDataBytes := msgData.([]byte)
		xlog.Error("bind body fail, request:[%s], err=%s", string(msgDataBytes), err.Error())
		xc.String(http.StatusOK, "")
		return
	}

	xc.Set(xng.KEY_PARAMS, req)
	xlog.Debug("received msg,channel=%d,type=%s", channel, req.MsgType)

	body, ok := xc.Get(gin.BodyBytesKey)
	if !ok {
		xlog.Error("get body fail")
		xc.String(http.StatusOK, "")
		return
	}
	bodyBytes := body.([]byte)
	bodystr := string(bodyBytes)
	xlog.Info("user req is, msgId:%v,form user:%v,to user :%v", req.MsgID, req.FromUserName, req.ToUserName)
	if !inGray(req.MsgType, req.FromUserName, channel) {
		xlog.Debug("send to php mobile")
		xc.Set("sendphp", true)
		channel := wxmsgService.WxidToChannel(req.ToUserName)
		if (req.Event == "subscribe" || req.Event == "unsubscribe") && (channel == user_message_center_api.ChannelXNGService || channel == user_message_center_api.ChannelXNGSubscribe) { //公众号消息于订阅号关注/取关消息额外投递一份到mq
			xngMsg, err := wxmsgService.NormalizeWxRawMsg(req)
			if err != nil {
				xc.String(http.StatusOK, "")
				return
			}
			if err = proxy.SendMessage(conf.NormalizedMsgTopic, "normalized_msg", xngMsg); err != nil {
				xlog.Error("doWxMsg.SendMessage fail,err=%s", err.Error())
				return
			}
			//jsonData, err := json.Marshal(xngMsg)
			//if err != nil {
			//	xlog.ErrorC(xc,"doWxMsg.marshal json fail，err=%s", err.Error())
			//	xc.String(http.StatusOK, "")
			//	return
			//}
			//if err = proxy.SendMsg(def.ProducerName, def.TopicNameNormalizedMsg, string(jsonData), "normalized_msg"); err != nil {
			//	xlog.ErrorC(xc,"doWxMsg.send msg fail,err=%s", err.Error())
			//	xc.String(http.StatusOK, "")
			//	return
			//}
		}
		if phpwxreceive.PhpWxReceive(bodystr, wxquery, channel, xc.ContentType()) {
			xc.String(http.StatusOK, "")
			return
		}
		xlog.Error("send to php mobile fail,body:%s,query info:%v", bodystr, *wxquery)
	}
	//post请求后面会带query参数，每次收到消息都校验，防止攻击
	if !wxmsgService.CheckWxSignature(wxquery, token) {
		xlog.Error("check signature fail")
		xc.String(http.StatusOK, "")
		return
	}
	Mid, err := userService.GetMidByChannel(req.ToUserName, req.FromUserName, req.MsgType, channel)
	if err != nil {
		xlog.Error("get mid fail,err :%v", err)
		return
	}
	mid := strconv.FormatInt(Mid, 10)
	xc.Set("mid", mid)
	xc.Set("sendphp", false)
	//var reqBytes []byte
	//if reqBytes, err = json.Marshal(req); err != nil {
	//	xlog.Error("json marshal fail")
	//	xc.String(http.StatusOK, "")
	//	return
	//}
	//if err = proxy.SendMsg("outer_callback", def.TopicNameWxRawMsg, string(reqBytes), ""); err != nil {
	//	xc.ReplyFail(lib.CodePara)
	//	return
	//}
	if err = proxy.SendMessage(conf.WXRawTopic, "", req); err != nil {
		xlog.Error("doWxMsg.SendMessage fail,err=%s, req=%v", err.Error(), req)
		return
	}
	xc.String(http.StatusOK, "")
}
