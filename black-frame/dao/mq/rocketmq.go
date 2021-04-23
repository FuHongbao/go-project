package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/conf"
)

const (
	RetryTime = 5
	SleepTime = time.Millisecond * 50
)

func NotifyByMq(ctx context.Context, topic, tag string, data interface{}) (err error) {
	msgData, err := json.Marshal(data)
	if err != nil {
		xlog.ErrorC(ctx, "fail to marshal data:%v, err:%v", data, err)
		return
	}

	msg := primitive.NewMessage(topic, msgData)
	msg.WithTag(tag)

	for i := 0; i < RetryTime; i++ {
		ret, errIgnore := conf.MqProducer.SendSync(context.Background(), msg)
		err = errIgnore
		if err != nil {
			xlog.ErrorC(ctx, "NotifyByMq fail to send msg, err:%v, topic:%s, tag:%s, try:%d, data:%v", err, topic, tag, i, msgData)
			time.Sleep(SleepTime)
			continue
		}
		if ret == nil {
			err = errors.New(fmt.Sprintf("NotifyByMq fail to send msg, ret is nil, topic:%s, tag:%s, try:%d, data:%v", topic, tag, i, msgData))
			continue
		}
		err = nil
		xlog.InfoC(ctx, "NotifyByMq success to send msg, topic:%s, tag:%s, data:%v, ret:%v", topic, tag, data, ret)
		break
	}
	return
}
