package app

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/conf"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/mq"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/service/appmsg"
	"xgit.xiaoniangao.cn/xngo/service/user_message_center_api"
)

//RcvXngAppMsg  小年糕app消息
func RcvXngAppMsg(c *gin.Context) {
	var (
		req    appmsg.AppRawMsg
		err    error
		xngMsg *user_message_center_api.RevReq
		//jsonData []byte
		xc = xng.NewXContext(c)
	)

	xlog.Debug("contenttype=%s", xc.ContentType())
	if err = xc.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		xlog.Error("bind body err, err=%s", err.Error())
		return
	}
	xngMsg, ret := appmsg.NormalizeAppRawMsg(&req)
	if ret != true {
		xc.ReplyFail(lib.CodePara)
		return
	}
	//if jsonData, err = json.Marshal(xngMsg); err != nil {
	//	xlog.Error("marshal json fail，err=%s", err.Error())
	//	xc.ReplyOKWithoutData() //给mq回复ok确认消费
	//	return
	//}
	if err = proxy.SendMessage(conf.NormalizedMsgTopic, "normalized_msg", xngMsg); err != nil {
		xlog.Error("RcvXngAppMsg.SendMessage fail,err=%s", err.Error())
		return
	}
	//if err = proxy.SendMsg(def.ProducerName, def.TopicNameNormalizedMsg, string(jsonData), "normalized_msg"); err != nil {
	//	xlog.Error("send msg fail,err=%s", err.Error())
	//	xc.ReplyOKWithoutData() //给mq回复ok确认消费
	//	return
	//}
	xc.ReplyOKWithoutData()
}
