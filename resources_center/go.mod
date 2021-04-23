module xgit.xiaoniangao.cn/xngo/service/resources_center

go 1.12

require (
	github.com/aliyun/alibaba-cloud-sdk-go v1.60.338
	github.com/aliyun/aliyun-oss-go-sdk v2.0.5+incompatible
	github.com/apache/rocketmq-client-go/v2 v2.1.0-rc5
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/garyburd/redigo v1.6.0
	github.com/gin-gonic/gin v1.6.2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.9.0
	github.com/robfig/cron v1.2.0
	github.com/satori/go.uuid v1.2.0
	github.com/spf13/cast v1.3.0
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo v1.1.0
	xgit.xiaoniangao.cn/xngo/lib/sdk v1.8.0
	xgit.xiaoniangao.cn/xngo/lib/utils v1.0.0
	xgit.xiaoniangao.cn/xngo/lib/xconf v1.0.2
	xgit.xiaoniangao.cn/xngo/lib/xconsul v1.1.2-0.20200709073444-1e3ef1251558
	xgit.xiaoniangao.cn/xngo/lib/xlog v1.6.0
	xgit.xiaoniangao.cn/xngo/lib/xmongo v1.0.5
	xgit.xiaoniangao.cn/xngo/lib/xnet v1.0.2
	xgit.xiaoniangao.cn/xngo/service/ids_api v1.0.1-0.20201104081956-103bc2a5e231
)

replace golang.org/x/time => github.com/golang/time v0.0.0-20190308202827-9d24e82272b4

replace golang.org/x/net => github.com/golang/net v0.0.0-20190227160552-c95aed5357e7

replace golang.org/x/crypto => github.com/golang/crypto v0.0.0-20190228050851-31a38585487a

replace golang.org/x/sys => github.com/golang/sys v0.0.0-20190228071610-92a0ff1e1e2f

replace golang.org/x/text => github.com/golang/text v0.3.2

replace golang.org/x/tools => github.com/golang/tools v0.0.0-20190520220859-26647e34d3c0

replace golang.org/x/sync => github.com/golang/sync v0.0.0-20190423024810-112230192c58
