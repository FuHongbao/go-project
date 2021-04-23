package sts

import (
	"github.com/gin-gonic/gin"
)

func InitRouters(app *gin.Engine) {
	group := app.Group("uptoken")
	group.POST("get", Get)
}
