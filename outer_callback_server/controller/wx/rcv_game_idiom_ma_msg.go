package wx

import (
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/service/user_message_center_api"
)

//ValidateGameidiomMa 校验小游戏消息签名
func ValidateGameidiomMa(c *gin.Context) {
	doValidate(c, user_message_center_api.ChannelGameIdiomMiniApp)
}

//RcvGameIdiomMaMsg 接收小游戏消息
func RcvGameIdiomMaMsg(c *gin.Context) {
	doWxMsg(c, user_message_center_api.ChannelGameIdiomMiniApp)
}
