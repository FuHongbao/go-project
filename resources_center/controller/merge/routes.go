package merge

import "github.com/gin-gonic/gin"

func InitRouters(app *gin.Engine) {
	group := app.Group("merge")
	group.POST("video", VideoMerge)        //合并视频资源
	group.POST("callback", ResultCallBack) //合并资源回调
}
