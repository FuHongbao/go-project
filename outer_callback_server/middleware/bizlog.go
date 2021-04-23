package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap/zapcore"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/conf"
	wxmsgService "xgit.xiaoniangao.cn/xngo/service/outer_callback_server/service/wxmsg"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/utils"
	"xgit.xiaoniangao.cn/xngo/service/user_message_center_api"
)

const (
	project = ""
	//XNG 小年糕平台
	XNG = "1"
	//XBD 小板凳平台
	XBD = "2"
	//TIA  tia平台
	TIA = "3"
	//GameIdiom 小游戏平台
	GameIdiom = "4"
	//GameDuet 小游戏平台
	GameDuet = "5"
)
const (
	platform = ""
	//WxMp 微信公众号
	WxMp = "1"
	//WxMa 微信小程序
	WxMa = "2"
	//WxSub 微信订阅号
	WxSub = "3"
	//IOS IOS
	IOS = "4"
	//Android Android
	Android = "5"
)

//BizLogCont 给数据组上传日志的结构体
type BizLogCont struct {
	WxRawMsg *wxmsgService.WxRawMsg
	Context  *xng.XContext
	APIName  string
	Mid      string
}
type addr struct {
	RemoteIP string
	LocalIP  string
}

//暂时不知道有啥用 暂且不删
func (a *addr) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("rmt", a.RemoteIP)
	enc.AddString("loc", a.LocalIP)
	return nil
}

//Bizlog 上报日志
func Bizlog() gin.HandlerFunc {
	return func(c *gin.Context) {
		URLPath := c.Request.URL.Path
		c.Next()
		params, exists := c.Get("params")
		sendPhp, _ := c.Get("sendphp")
		id, ok := c.Get("mid")
		isSendPhp, value := sendPhp.(bool)
		if !value || isSendPhp {
			return
		}

		if (!exists || !ok) && !isSendPhp {
			xlog.Error("get context fail")
			return
		}
		path := ""
		if len(URLPath) > 1 {
			path = URLPath[1:len(URLPath)]
		}
		request, ok := (params).(*wxmsgService.WxRawMsg)
		if !ok || request == nil {
			xlog.Error("trans struct xxRawMsg form context map fail")
			return
		}
		bizLogCont := &BizLogCont{}
		bizLogCont.WxRawMsg = request
		bizLogCont.APIName = path
		bizLogCont.Mid = utils.InterfaceToString(id)
		bizLogCont.Context = xng.NewXContext(c)
		SendBizSrvLog(bizLogCont)
	}
}

//SendBizSrvLog 日志通过InfoW 写入数据组指定的路径
func SendBizSrvLog(bizLogCont *BizLogCont) {
	var codever string
	if bizLogCont.Context == nil || bizLogCont.Context.Request == nil {
		xlog.Error("get context or context request fail")
		return
	}

	if bizLogCont.Context.Request.Header == nil {
		xlog.Error("get request header fail")
		return
	}
	clientIP := bizLogCont.Context.ClientIP()
	uastr := bizLogCont.Context.GetHeader("User-Agent")
	if uastr == "" || clientIP == "" {
		xlog.Error("get user agent or ip fail")
		return
	}
	wxid := bizLogCont.WxRawMsg.ToUserName
	openid := bizLogCont.WxRawMsg.FromUserName
	msgType := bizLogCont.WxRawMsg.MsgType
	if wxid == "" || openid == "" || msgType == "" {
		xlog.Error("get  wxid or openid or msgTypeis  nil")
		return
	}
	//code_ver = getAppVersion(bizLogCont.Context)
	if bizLogCont.Context.Request.Method == "POST" {
		channel := wxmsgService.WxidToChannel(wxid)
		pj := getPjByChannel(channel)
		pf := getPfByChannel(channel)

		if Android == pf || IOS == pf {
			codever = getAppVersion(bizLogCont.Context)
		} else {
			codever = ""
		}
		jsonstr := map[string]interface{}{
			"date": time.Now().Format("2006-01-02 15:04:05"),
			"stat": map[string]interface{}{
				"t":   time.Now().UnixNano() / 1000000,
				"pf":  pf,
				"pj":  pj,
				"uid": openid,
				"v":   codever,
				"mid": bizLogCont.Mid,
				"e": []map[string]interface{}{
					{
						"ac": "api",
						"t":  time.Now().UnixNano() / 1000000,
						"data": map[string]interface{}{
							"name":     bizLogCont.APIName,
							"msg_type": msgType,
						},
					},
				},
				"ext": map[string]interface{}{},
			},
			"ua": uastr,
			"ip": clientIP,
		}
		conf.Bizlog.InfoW("", jsonstr)
	}

}
func getPjByChannel(channel int) string {
	switch channel {
	case user_message_center_api.ChannelXNGService, user_message_center_api.ChannelXNGMiniApp, user_message_center_api.ChannelXNGAndroid, user_message_center_api.ChannelXNGIOS, user_message_center_api.ChannelXNGSubscribe:
		return XNG
	case user_message_center_api.ChannelXBDService, user_message_center_api.ChannelXBDSubscribe, user_message_center_api.ChannelXBDMiniApp:
		return XBD
	case user_message_center_api.ChannelTIAService:
		return TIA
	case user_message_center_api.ChannelGameIdiomMiniApp:
		return GameIdiom
	case user_message_center_api.ChannelGameDuetMiniApp:
		return GameDuet
	default:
		xlog.Debug("get pj by channel fail,channel:%v", channel)
		return project

	}
}
func getPfByChannel(channel int) string {
	switch channel {
	case user_message_center_api.ChannelXBDService, user_message_center_api.ChannelTIAService, user_message_center_api.ChannelXNGService:
		return WxMp
	case user_message_center_api.ChannelXNGMiniApp, user_message_center_api.ChannelXBDMiniApp, user_message_center_api.ChannelGameIdiomMiniApp, user_message_center_api.ChannelGameDuetMiniApp:
		return WxMa
	case user_message_center_api.ChannelXNGSubscribe, user_message_center_api.ChannelXBDSubscribe:
		return WxSub
	case user_message_center_api.ChannelXNGIOS:
		return IOS
	case user_message_center_api.ChannelXNGAndroid:
		return Android
	default:
		xlog.Debug("get pf by channel fail,channel:%v", channel)
		return platform
	}
}

func getAppVersion(context *xng.XContext) (appVersion string) {
	appVersion = context.Request.Header.Get("H-Av")
	return appVersion

}
