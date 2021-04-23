package wx

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/def"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/mq"
	wxmsgService "xgit.xiaoniangao.cn/xngo/service/outer_callback_server/service/wxmsg"
	"xgit.xiaoniangao.cn/xngo/service/user_message_center_api"
)

//RcvMqWxMsg 除小年糕平台之外的消息
func RcvMqWxMsg(c *gin.Context) {
	var (
		req      wxmsgService.WxRawMsg
		err      error
		xngMsg   *user_message_center_api.RevReq
		jsonData []byte
		xc       = xng.NewXContext(c)
	)

	xlog.Debug("contenttype=%s", xc.ContentType())
	if err = xc.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		xlog.Error("bind body err, err=%s", err.Error())
		xc.ReplyOKWithoutData() //给mq回复ok确认消费
		return
	}
	xc.Set(xng.KEY_PARAMS, &req)

	xngMsg, err = wxmsgService.NormalizeWxRawMsg(&req)
	if err != nil {
		xlog.Debug("NormalizeWxRawMsg failed, msg[%v]", req)
		xc.ReplyOKWithoutData() //处理消息异常
		return
	}

	if jsonData, err = json.Marshal(xngMsg); err != nil {
		xlog.Error("marshal json fail，err=%s", err.Error())
		xc.ReplyOKWithoutData() //给mq回复ok确认消费
		return
	}
	if err = proxy.SendMsg(def.ProducerName, def.TopicNameNormalizedMsg, string(jsonData), "normalized_msg"); err != nil {
		xlog.Error("send msg fail,err=%s", err.Error())
		xc.ReplyOKWithoutData() //给mq回复ok确认消费
		return
	}

	xc.ReplyOKWithoutData()
}
