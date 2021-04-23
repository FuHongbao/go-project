package trans

import "github.com/gin-gonic/gin"

func InitRouters(app *gin.Engine) {
	group := app.Group("trans")
	group.POST("album_trans", AlbumTransToMp4)
	group.POST("video_mp4_trans", VideoTransToMp4)
	group.POST("callback", CallbackForTrans)
}
