package xprometheus

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//InitRouter 注册路由
func InitRouter(app *gin.Engine) {
	group := app.Group("")
	group.Any("/metrics", func(c *gin.Context) {
		h := promhttp.Handler()
		h.ServeHTTP(c.Writer, c.Request)
	})
}
