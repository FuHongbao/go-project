package snap

import "github.com/gin-gonic/gin"

func InitRouters(app *gin.Engine) {
	group := app.Group("snap")
	group.POST("album_snap", AlbumSnapShot)
	group.POST("video_frame_list", VideoFrameList)
	group.POST("album_frame_list", AlbumFrameList)
	group.POST("replace_snap", ReplaceSnapShot)
}
