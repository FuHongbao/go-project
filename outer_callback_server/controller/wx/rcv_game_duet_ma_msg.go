package wx

import (
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/service/user_message_center_api"
)

//ValidateGameidiomMa 校验对唱小程序消息签名
func ValidateGameDuetMa(c *gin.Context) {
	doValidate(c, user_message_center_api.ChannelGameDuetMiniApp)
}

//RcvGameIdiomMaMsg 接收对唱小程序消息
func RcvGameDuetMaMsg(c *gin.Context) {
	doWxMsg(c, user_message_center_api.ChannelGameDuetMiniApp)
}
