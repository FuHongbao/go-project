package wx

import (
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/service/user_message_center_api"
)

//ValidateXngMa 校验小年糕小程序消息签名
func ValidateXngMa(c *gin.Context) {
	doValidate(c, user_message_center_api.ChannelXNGMiniApp)
}

//RcvXngMaMsg 接收小年糕小程序消息
func RcvXngMaMsg(c *gin.Context) {
	doWxMsg(c, user_message_center_api.ChannelXNGMiniApp)
}
