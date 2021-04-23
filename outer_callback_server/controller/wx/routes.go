package wx

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/conf"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/controller/app"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/middleware"
)

func init() {
	conf.Gin.Any("", ping)
	conf.Gin.Any("ping", ping)

	group := conf.Gin.Group("wx")
	group.GET("rcv_xng_mp_msg", ValidateXngMp)
	group.POST("rcv_xng_mp_msg", middleware.Bizlog(), RcvXngMpMsg)

	group.GET("rcv_xng_ma_msg", ValidateXngMa)
	group.POST("rcv_xng_ma_msg", middleware.Bizlog(), RcvXngMaMsg)

	group.GET("rcv_xng_sub_msg", ValidateXngSub)
	group.POST("rcv_xng_sub_msg", middleware.Bizlog(), RcvXngSubMsg)

	group.POST("rcv_mq_wx_msg", RcvMqWxMsg)
	group.POST("rcv_mq_wx_media_upload_ok_msg", RcvMqWxMediaUploadOkMsg)

	group.GET("rcv_game_idiom_ma_msg", ValidateGameidiomMa)
	group.POST("rcv_game_idiom_ma_msg", middleware.Bizlog(), RcvGameIdiomMaMsg)

	group.GET("rcv_duet_ma_msg", ValidateGameDuetMa)                     //对唱小程序校验
	group.POST("rcv_duet_ma_msg", middleware.Bizlog(), RcvGameDuetMaMsg) //接收对唱小程序消息

	group2 := conf.Gin.Group("app")
	group2.POST("rcv_xng_app_msg", middleware.Bizlog(), app.RcvXngAppMsg)

	conf.Gin.Any("health", health)
}

func ping(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}
func health(c *gin.Context) {
	c.String(http.StatusOK, "SUCCESS")
}
