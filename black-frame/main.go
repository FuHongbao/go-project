package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/utils"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/conf"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/controller/frame"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/middleware"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/queue"
)

func main() {
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(xng.Boss(conf.C.App.Name, xlog.GetDefaultLogger(), nil))
	engine.Use(middleware.Cors())
	engine.Use(xng.RetTranlate(func(code lib.Code) lib.ServiceLevel {
		switch code {
		case lib.CodeSrv:
			return lib.LevelError
		case lib.CodePara:
			return lib.LevelWarning
		default:
			return lib.LevelNormal
		}
	}))
	queue.StartQueue()
	initRouter(engine)
	addr := fmt.Sprintf("%s:%d", "0.0.0.0", conf.C.App.Port)
	xlog.Info("listen address:%s", addr)
	err := utils.ListenAndServe(addr, engine)
	if err != nil {
		xlog.Fatal("listen address:%s, err:%v", addr, err)
	}
}
func initRouter(e *gin.Engine) {
	e.Any("/health", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "SUCCESS")
	})
	frame.InitRouters(e)
}
