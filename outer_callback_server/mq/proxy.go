package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"io/ioutil"
	"strings"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/lib/xnet/xhttp"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/conf"
)

var proxyURL = conf.C.MQ["default"].NameSrvAddr
var client *xhttp.Client

//Resp defined
type Resp struct {
	Ret    int         `json:"ret"`
	Data   interface{} `json:"data"`
	Msg    string      `json:"msg,omitempty"`
	Detail string      `json:"detail,omitempty"`
}

//SendMsg 向指定的topic 发送消息
func SendMsg(producerName, topic, body, tag string) (err error) {
	xlog.Info("producerName=%s,topic=$s,body=%s", producerName, topic, body)

	url := fmt.Sprintf("http://%s/send?producerName=%s&topic=%s", proxyURL, producerName, topic)
	if tag != "" {
		url = fmt.Sprintf("%s&tag=%s", url, tag)
	}
	resp, err := client.Post(url, "text/plain", strings.NewReader(body))
	if err != nil {
		xlog.Error("Send Msg Failed! ProducerName: %s, Topic: %s, URL: %s, Err: %v", producerName, topic, url, err)
		xlog.Debug("Body: %s", body)
		return
	}
	defer resp.Body.Close()
	var ret Resp
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		xlog.Error("Read Response Body Failed")
		return
	}
	err = json.Unmarshal(respBody, &ret)
	if err != nil {
		xlog.Error("Parse Response Body Failed, body: %s", string(respBody))
		return
	}
	if ret.Ret == 1 {
		xlog.Info("Send Msg Ok, Topic: %v, MsgId: %v", topic, ret.Data)
	} else {
		xlog.Error("Send Msg Err, Topic: %v, Ret: %v", topic, ret)
	}
	return
}
func init() {
	client = xhttp.NewClient()
	client.Retries = 1
	client.Timeout = time.Second * 3
}

func SendMessage(topic, tag string, data interface{}) (err error) {
	msgData, err := json.Marshal(data)
	if err != nil {
		xlog.Error("failed to marshal data: %+v, err: %v", data, err)
		return err
	}

	msg := primitive.NewMessage(topic, msgData)
	msg.WithTag(tag)
	sendResult, err := conf.MqProducer.SendSync(context.Background(), msg)
	if err != nil {
		xlog.Error("failed to send message, err: %v, topic:%s, data: %v", err, topic, msgData)
		return err
	}
	xlog.Info("success to send message, topic: %s, data: %v, send_result: %+v", topic, data, sendResult)
	return nil
}
