module xgit.xiaoniangao.cn/xngo/service/outer_callback_server

go 1.12

replace golang.org/x/time => github.com/golang/time v0.0.0-20190308202827-9d24e82272b4

replace golang.org/x/net => github.com/golang/net v0.0.0-20190227160552-c95aed5357e7

replace golang.org/x/crypto => github.com/golang/crypto v0.0.0-20190228050851-31a38585487a

replace golang.org/x/sys => github.com/golang/sys v0.0.0-20190228071610-92a0ff1e1e2f

replace golang.org/x/text => github.com/golang/text v0.3.2

replace golang.org/x/tools => github.com/golang/tools v0.0.0-20190520220859-26647e34d3c0

replace golang.org/x/sync => github.com/golang/sync v0.0.0-20190423024810-112230192c58

require (
	github.com/apache/rocketmq-client-go/v2 v2.0.0
	github.com/garyburd/redigo v1.6.0
	github.com/gin-gonic/gin v1.3.0
	go.uber.org/zap v1.10.0
	xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo v1.1.0
	xgit.xiaoniangao.cn/xngo/lib/lc v1.1.0
	xgit.xiaoniangao.cn/xngo/lib/sdk v1.3.27
	xgit.xiaoniangao.cn/xngo/lib/utils v1.0.0
	xgit.xiaoniangao.cn/xngo/lib/xconf v1.0.2
	xgit.xiaoniangao.cn/xngo/lib/xlog v1.6.0
	xgit.xiaoniangao.cn/xngo/lib/xmongo v1.0.5
	xgit.xiaoniangao.cn/xngo/lib/xnet v1.0.2
	xgit.xiaoniangao.cn/xngo/service/ids_api v1.0.0
	xgit.xiaoniangao.cn/xngo/service/user_message_center_api v1.1.2
)
