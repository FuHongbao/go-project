package main

import (
	"context"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

const (
	TopicNotifyXng = "topic_upload_xng"
	ResSrcWXmine   = "11"
)

// 回调处理函数
func callbackSuccess(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for i := range msgs {
		// todo 一些业务处理逻辑
		fmt.Println("message: ", msgs[i])
	}
	return consumer.ConsumeSuccess, nil
}

func main() {
	mqConsumer, err := rocketmq.NewPushConsumer(
		consumer.WithGroupName("notify_xng_upload"),
		// 注册线上mq
		consumer.WithNameServer([]string{"ip:port"}),
		consumer.WithConsumerModel(consumer.Clustering),
	)
	if err != nil {
		panic(fmt.Sprintf("create rocket mq consumer err:%v", err))
		return
	}
	tag := fmt.Sprintf("src_%s", ResSrcWXmine)

	// 根据tag筛选
	selector := consumer.MessageSelector{
		Type:       consumer.TAG,
		Expression: tag,
	}

	// 订阅 "topic_upload_xng" topic
	err = mqConsumer.Subscribe("topic_upload_xng", selector, callbackSuccess)
	if err != nil {
		panic(fmt.Sprintf("subscribe rocket mq consumer err:%v", err))
		return
	}
	fmt.Println("start")
	err = mqConsumer.Start()
	if err != nil {
		panic(fmt.Sprintf("start rocket mq consumer err:%v", err))
		return
	}
	defer mqConsumer.Shutdown()
	fmt.Println("over")
}
