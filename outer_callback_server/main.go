package main

import (
	"fmt"
	"xgit.xiaoniangao.cn/xngo/lib/utils"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/conf"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/controller/mq"
	_ "xgit.xiaoniangao.cn/xngo/service/outer_callback_server/controller/wx"
)

func main() {
	if conf.CheckConfig {
		return
	}
	if conf.C.MqGo.Switch == "on" {
		mq.SubScribe()
	}
	addr := fmt.Sprintf("%s:%d", "0.0.0.0", conf.C.App.Port)
	err := utils.ListenAndServe(addr, conf.Gin)
	if err != nil {
		xlog.Error("err: %v, addr: %v", err, addr)
	}
	conf.ShutDownMq()
}
