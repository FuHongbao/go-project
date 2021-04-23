package conf

import (
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/apache/rocketmq-client-go/v2/rlog"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
)

var (
	MqProducer rocketmq.Producer
	// 外部第三方服务回调服务消息队列
	WXRawConsumer        rocketmq.PushConsumer
	UploadResultConsumer rocketmq.PushConsumer
)

const (
	WXRawConsumerGroupName        = "wx_raw_msg_consumer"
	WXRawTopic                    = "wx_raw_msg"
	UploadResultConsumerGroupName = "upload_media_result_consumer"
	UploadResultTopic             = "upload_wx_media_ok"
	MyProducerName                = "outer_callback"
	NormalizedMsgTopic            = "normalized_msg"

	MqRetryTimes = 3
)

type MqConf struct {
	NameServers []string `mapstructure:"nameServers"`
	Switch      string   `mapstructure:"switch"`
}

func InitMq(mqConf MqConf, mqlog *xlog.Config) {
	//lc := &xlog.Config{FileConfig: &xlog.FileConfig{
	//	LogFilePath: "./log/mq_",
	//	MaxSize:     1024,
	//	MaxBackups:  5,
	//	MaxAge:      7,
	//	Console:     false,
	//	LevelString: "info",
	//}}
	logger := xlog.NewLogger(mqlog, C.App.Name)
	l := &MQLog{
		logger,
	}
	rlog.SetLogger(l)
	var err error
	MqProducer, err = rocketmq.NewProducer(
		producer.WithGroupName(MyProducerName),
		producer.WithNameServer(mqConf.NameServers),
		producer.WithRetry(MqRetryTimes),
	)
	if err != nil {
		xlog.Fatal("init rocket mq producer err: %v", err)
		return
	}
	err = MqProducer.Start()
	if err != nil {
		xlog.Fatal("producer mq start err: %v", err)
		return
	}
	xlog.Info("MqProducer init success")
	initConsumer(mqConf.NameServers)
}
func initConsumer(c []string) {
	var err error
	WXRawConsumer, err = rocketmq.NewPushConsumer(
		consumer.WithGroupName(WXRawConsumerGroupName),
		consumer.WithNameServer(c),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithConsumeMessageBatchMaxSize(1),
	)
	if err != nil {
		xlog.Fatal("init WXRawConsumer err: %v", err)
		return
	}
	xlog.Info("WXRawConsumer init success")
	UploadResultConsumer, err = rocketmq.NewPushConsumer(
		consumer.WithGroupName(UploadResultConsumerGroupName),
		consumer.WithNameServer(c),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithConsumeMessageBatchMaxSize(1),
	)
	if err != nil {
		xlog.Fatal("init WXRawConsumer err: %v", err)
		return
	}
	xlog.Info("UploadResultConsumer init success")
}

func ShutDownMq() {
	_ = MqProducer.Shutdown()
	_ = WXRawConsumer.Shutdown()
	_ = UploadResultConsumer.Shutdown()
}

type MQLog struct {
	logger *xlog.XLogger
}

func getFormatArgs(msg string, fields map[string]interface{}) (format string, args []interface{}) {
	format = "msg:%s"
	args = []interface{}{msg}
	for k, v := range fields {
		format = format + ", " + k + ":%v"
		args = append(args, v)
	}
	return
}

func (l *MQLog) Debug(msg string, fields map[string]interface{}) {
	if msg == "" && len(fields) == 0 {
		return
	}
	format, args := getFormatArgs(msg, fields)
	l.logger.Debug(format, args...)
}

func (l *MQLog) Info(msg string, fields map[string]interface{}) {
	if msg == "" && len(fields) == 0 {
		return
	}
	format, args := getFormatArgs(msg, fields)
	l.logger.Info(format, args...)
}

func (l *MQLog) Warning(msg string, fields map[string]interface{}) {
	if msg == "" && len(fields) == 0 {
		return
	}
	format, args := getFormatArgs(msg, fields)
	l.logger.Warn(format, args...)
}

func (l *MQLog) Error(msg string, fields map[string]interface{}) {
	if msg == "" && len(fields) == 0 {
		return
	}
	format, args := getFormatArgs(msg, fields)
	l.logger.Error(format, args...)
}

func (l *MQLog) Fatal(msg string, fields map[string]interface{}) {
	if msg == "" && len(fields) == 0 {
		return
	}
	format, args := getFormatArgs(msg, fields)
	l.logger.Fatal(format, args...)
}
