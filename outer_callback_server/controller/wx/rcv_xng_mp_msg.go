package wx

import (
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/service/user_message_center_api"
)

//ValidateXngMp 校验小年糕服务号消息签名
func ValidateXngMp(c *gin.Context) {
	doValidate(c, user_message_center_api.ChannelXNGService)
}

//RcvXngMpMsg 接收小年糕服务号消息
func RcvXngMpMsg(c *gin.Context) {
	doWxMsg(c, user_message_center_api.ChannelXNGService)
}
