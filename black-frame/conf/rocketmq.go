package conf

import (
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
)

type MqConf struct {
	NameServers []string `mapstructure:"nameServers"`
}

var (
	MqProducer rocketmq.Producer
	MqConsumer rocketmq.PushConsumer
)

const (
	MqRetryTimes = 3
)

func InitMq(mqConf *MqConf) {
	if mqConf == nil {
		xlog.Fatal("mq config is nil")
	}
	xlog.Debug("mq config, mq:%v, nameservers:%v", mqConf, mqConf.NameServers)
	var err error
	MqProducer, err = rocketmq.NewProducer(
		producer.WithGroupName("produce_black_frames"),
		producer.WithNameServer(mqConf.NameServers),
		producer.WithRetry(MqRetryTimes),
	)
	if err != nil {
		xlog.Fatal("init rocket mq producer(notify_black_frames) err:%v", err)
	}
	MqConsumer, err = rocketmq.NewPushConsumer(
		consumer.WithGroupName("consume_black_frames"),
		consumer.WithNameServer(mqConf.NameServers),
	)
	if err != nil {
		xlog.Fatal("init rocket mq consumer(notify_black_frames) err:%v", err)
	}
	err = MqProducer.Start()
	if err != nil {
		xlog.Fatal("producer(notify_black_frames) mq start err:%v", err)
		return
	}
}

func ShutDownMq() {
	_ = MqProducer.Shutdown()
}
