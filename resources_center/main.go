package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron"
	"net/http"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/utils"
	"xgit.xiaoniangao.cn/xngo/lib/xconsul/xagent"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/controller/auth"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/controller/merge"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/controller/resource"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/controller/snap"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/controller/speech"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/controller/sts"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/controller/trans"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/controller/url"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/controller/video"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/controller/xprometheus"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/cron_task"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/middleware"
)

func main() {
	if conf.CheckConfig {
		return
	}

	// gin engine实例
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
	initRouter(engine)
	//go regConsulAgentService()
	cron_task.RemoteAliClient()
	//定时任务每日凌晨1点执行一次
	c := cron.New()
	//err := c.AddFunc("0 */1 * * * ?", cron_task.RegularCleanMultiParts) //每分钟一次
	err := c.AddFunc("0 0 1 * * ?", cron_task.RegularCleanMultiParts)
	if err != nil {
		xlog.Fatal("cron add func:RegularCleanMultiParts, err:%v", err)
		return
	}
	c.Start()
	addr := fmt.Sprintf("%s:%d", "0.0.0.0", conf.C.App.Port)
	xlog.Info("listen address:%s", addr)
	err = utils.ListenAndServe(addr, engine)
	if err != nil {
		xlog.Fatal("listen address:%s, err:%v", addr, err)
	}
	conf.ShutDownMq()
}

func initRouter(e *gin.Engine) {
	e.Any("/health", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "SUCCESS")
	})
	video.InitRouters(e)
	url.InitRoutes(e)
	sts.InitRouters(e)
	resource.InitRouters(e)
	auth.InitRouters(e)
	snap.InitRouters(e)
	merge.InitRouters(e)
	trans.InitRouters(e)
	speech.InitRouters(e)
	xprometheus.InitRouter(e)
}

//注册 consul
func regConsulAgentService() {
	ipAddress, err := xagent.GetLocalIPAddress()
	if err != nil {
		xlog.Error("get local ip err:%v", err)
		return
	}
	serviceConf := xagent.DefaultServiceConf("BusinessExporter", ipAddress, conf.C.App.Port)

	serviceName := conf.C.App.Name
	serviceConf.Tags = append(serviceConf.Tags, fmt.Sprint("serviceName=", serviceName))
	if err = xagent.RegisterService(serviceConf); err != nil {
		xlog.Error("reg consul agent service err:%v", err)
		return
	}
	xlog.Info("reg consul agent service done!")
}
