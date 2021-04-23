package speech

import (
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/speech"
)

type RespSpeechToken struct {
	AppKey string `json:"appkey"`
	Token  string `json:"token"`
	Url    string `json:"url"`
	Host   string `json:"host"`
}

func GetSpeechToken(c *gin.Context) {
	xc := xng.NewXContext(c)
	token, err := speech.GetSpeechToken(xc)
	if err != nil {
		xlog.ErrorC(xc, "GetSpeechToken.GetSpeechToken failed, err:[%v]", err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resp := RespSpeechToken{
		AppKey: conf.C.Speech.AppKey,
		Token:  token,
		Url:    conf.C.Speech.Url,
		Host:   conf.C.Speech.Host,
	}
	xc.ReplyOK(resp)
}
