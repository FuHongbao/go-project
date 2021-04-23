package wx

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gin-gonic/gin/json"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/def"
	proxy "xgit.xiaoniangao.cn/xngo/service/outer_callback_server/mq"
)

//UploadMsgReq defined
type UploadMsgReq struct {
	MsgID string `json:"msg_id" bson:"msg_id"`
	Qid   int64  `json:"qid" bson:"qid"`
}

//RcvMqWxMediaUploadOkMsg 接收图片下载成功的消息
func RcvMqWxMediaUploadOkMsg(c *gin.Context) {
	var (
		jsonData []byte
		err      error
		req      = &UploadMsgReq{}
		xc       = xng.NewXContext(c)
	)

	xlog.Debug("RcvMqWxMediaUploadOkMsg")

	if err = c.ShouldBindBodyWith(req, binding.JSON); err != nil {
		xlog.Error("bind body fail,err=%s", err)
		xc.ReplyOKWithoutData() //给mq发送确认
		return
	}
	c.Set(xng.KEY_PARAMS, req)

	if jsonData, err = json.Marshal(req); err != nil {
		xlog.Error("marshal json fail,err=%s", err.Error())
		xc.ReplyOKWithoutData() //给mq发送确认
		return
	}
	if err = proxy.SendMsg(def.ProducerName, def.TopicNameNormalizedMsg, string(jsonData), "update_msg"); err != nil {
		xlog.Error("send msg fail,err=%s", err.Error())
		xc.ReplyOKWithoutData() //给mq发送确认
		return
	}

	xc.ReplyOKWithoutData()
}
