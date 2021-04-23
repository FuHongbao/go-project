package conf

import (
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
)

type MqConf struct {
	NameServers []string `mapstructure:"nameServers"`
}

var (
	MqProducer rocketmq.Producer
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
		producer.WithGroupName("notify_upload"),
		producer.WithNameServer(mqConf.NameServers),
		producer.WithRetry(MqRetryTimes),
	)
	if err != nil {
		xlog.Fatal("init rocket mq producer(notify_upload) err:%v", err)
	}

	err = MqProducer.Start()
	if err != nil {
		xlog.Fatal("producer(notify_upload) mq start err:%v", err)
		return
	}

}

func ShutDownMq() {
	_ = MqProducer.Shutdown()
}
