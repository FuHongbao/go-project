package mq

import (
	"context"
	"encoding/json"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/conf"
	proxy "xgit.xiaoniangao.cn/xngo/service/outer_callback_server/mq"
	wxmsgService "xgit.xiaoniangao.cn/xngo/service/outer_callback_server/service/wxmsg"
	"xgit.xiaoniangao.cn/xngo/service/user_message_center_api"
)

func handleWxRawMsg(xc context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for _, msg := range msgs {
		var (
			req    wxmsgService.WxRawMsg
			err    error
			xngMsg *user_message_center_api.RevReq
		)
		err = json.Unmarshal(msg.Body, &req)
		if err != nil {
			xlog.Error("handleWxRawMsg.Unmarshal param err: %v , req : %v", err, req)
			return consumer.ConsumeSuccess, err
		}
		xlog.Debug("handleWxRawMsg.revc msg:[%v]", req)
		xngMsg, err = wxmsgService.NormalizeWxRawMsg(&req)
		if err != nil {
			xlog.Error("handleWxRawMsg.NormalizeWxRawMsg err: %v , req : %v", err, req)
			return consumer.ConsumeSuccess, err
		}
		xlog.Debug("handleWxRawMsg.NormalizeWxRawMsg msg:[%v]", xngMsg)
		if err = proxy.SendMessage(conf.NormalizedMsgTopic, "normalized_msg", xngMsg); err != nil {
			xlog.Error("handleWxRawMsg.SendMessage fail,err=%s", err.Error())
			return consumer.ConsumeRetryLater, err
		}
	}
	return consumer.ConsumeSuccess, nil
}

//UploadMsgReq defined
type UploadMsgReq struct {
	MsgID string `json:"msg_id" bson:"msg_id"`
	Qid   int64  `json:"qid" bson:"qid"`
}

func handleUploadMediaResult(xc context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for _, msg := range msgs {
		var (
			err error
			req = &UploadMsgReq{}
		)
		err = json.Unmarshal(msg.Body, &req)
		if err != nil {
			xlog.Error("handleUploadMediaResult.Unmarshal param err: %v , req : %v", err, req)
			return consumer.ConsumeSuccess, err
		}
		xlog.Debug("handleUploadMediaResult.Unmarshal, topic:[%v], tag:update_msg, req", conf.NormalizedMsgTopic, req)
		if err = proxy.SendMessage(conf.NormalizedMsgTopic, "update_msg", req); err != nil {
			xlog.Error("handleUploadMediaResult.send msg fail,err=%s", err.Error())
			return consumer.ConsumeRetryLater, err
		}
	}
	return consumer.ConsumeSuccess, nil
}

func SubScribe() {
	err := conf.WXRawConsumer.Subscribe(conf.WXRawTopic, consumer.MessageSelector{}, handleWxRawMsg)
	if err != nil {
		xlog.Fatal("consumer subscribe topic: %s, failed, err: %v", conf.WXRawTopic, err)
		return
	}
	err = conf.WXRawConsumer.Start()
	if err != nil {
		xlog.Fatal("WXRawConsumer mq start failed, err: %v", err)
		return
	}
	err = conf.UploadResultConsumer.Subscribe(conf.UploadResultTopic, consumer.MessageSelector{}, handleUploadMediaResult)
	if err != nil {
		xlog.Fatal("consumer subscribe topic: %s, failed, err: %v", conf.UploadResultTopic, err)
		return
	}
	err = conf.UploadResultConsumer.Start()
	if err != nil {
		xlog.Fatal("UploadResultConsumer mq start failed, err: %v", err)
		return
	}
	xlog.Info("Starting to consume message success!")
}
