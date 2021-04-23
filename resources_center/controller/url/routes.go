package url

import "github.com/gin-gonic/gin"

func InitRoutes(e *gin.Engine) {
	group := e.Group("url")
	group.POST("img", ImgURL)
	group.POST("video", VideoURL)
	group.POST("album", AlbumURL)
	group.POST("lyric", LyricURL)
	group.POST("static", StaticURL)
	group.POST("music", MusicURL)
	group.POST("static_img", StaticImgURL)
	group.POST("live_video", LiveVideoURL)
	group.POST("live_guide_video", LiveGuideVideoURL)
	group.POST("audit_music", AuditMusicURL)
	group.POST("app_log", AppLogURL)
	group.POST("album_m3u8_exp", AlbumM3u8ExpURL)

	g2 := e.Group("url/v2")
	g2.POST("img", ImgURLV2)
}
