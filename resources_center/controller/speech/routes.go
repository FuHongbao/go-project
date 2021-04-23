package speech

import "github.com/gin-gonic/gin"

func InitRouters(app *gin.Engine) {
	group := app.Group("speech")
	group.POST("token", GetSpeechToken)
}
