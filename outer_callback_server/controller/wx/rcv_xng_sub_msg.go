package wx

import (
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/service/user_message_center_api"
)

//ValidateXngSub 校验小年糕订阅号消息签名
func ValidateXngSub(c *gin.Context) {
	doValidate(c, user_message_center_api.ChannelXNGSubscribe)
}

//RcvXngSubMsg 接收小年糕订阅号消息
func RcvXngSubMsg(c *gin.Context) {
	doWxMsg(c, user_message_center_api.ChannelXNGSubscribe)
}
