module xgit.xiaoniangao.cn/xngo/service/black-frame

go 1.12

require (
	github.com/apache/rocketmq-client-go/v2 v2.1.0-rc5
	github.com/garyburd/redigo v1.6.0
	github.com/gin-gonic/gin v1.6.2
	github.com/kr/pretty v0.1.0 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo v1.1.0
	xgit.xiaoniangao.cn/xngo/lib/sdk v1.8.0
	xgit.xiaoniangao.cn/xngo/lib/utils v1.0.0
	xgit.xiaoniangao.cn/xngo/lib/xconf v1.0.2
	xgit.xiaoniangao.cn/xngo/lib/xconsul v1.1.2-0.20200709073444-1e3ef1251558
	xgit.xiaoniangao.cn/xngo/lib/xlog v1.6.0
	xgit.xiaoniangao.cn/xngo/lib/xnet v1.0.2
	xgit.xiaoniangao.cn/xngo/service/ids_api v1.0.1-0.20201104081956-103bc2a5e231
)

replace golang.org/x/sys => github.com/golang/sys v0.0.0-20190405154228-4b34438f7a67
