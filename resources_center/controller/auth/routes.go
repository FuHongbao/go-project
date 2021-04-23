package auth

import (
	"github.com/gin-gonic/gin"
)

func InitRouters(app *gin.Engine) {
	group := app.Group("authorize")
	group.POST("get_multi_author", GetMultiPartAuth) //获取分片上传签名授权
}
