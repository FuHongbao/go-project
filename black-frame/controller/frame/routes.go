package frame

import (
	"github.com/gin-gonic/gin"
)

func InitRouters(app *gin.Engine) {
	group := app.Group("frame")
	group.POST("del_black_frame", DelBlackFrame)
}
