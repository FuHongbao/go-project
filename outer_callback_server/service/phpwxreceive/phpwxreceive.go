package phpwxreceive

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/conf"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/service/wxmsg"
	"xgit.xiaoniangao.cn/xngo/service/user_message_center_api"
)

//PhpWxReceive 调用php接收消息的接口
func PhpWxReceive(rawmsg string, wxquery *wxmsg.WxQuery, channel int, contentType string) bool {
	startTime := time.Now()
	url := getPhpURL(channel)
	if url == "" {
		xlog.Error("php receive url empty")
		return false
	}
	url = fmt.Sprintf("%s?signature=%s&timestamp=%s&nonce=%s&echostr=%s", url, wxquery.Signature, wxquery.Timestamp, wxquery.Nonce, wxquery.Echostr)

	xlog.Debug("url=%s", url)

	resp, err := http.Post(url, contentType, strings.NewReader(rawmsg))
	if err != nil {
		xlog.Error("call php receive fail,err=%s", err.Error())
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		xlog.Error("call php receive fail,status code=%d", resp.StatusCode)
		return false
	}

	body, err := ioutil.ReadAll(resp.Body)
	xlog.Info("php resp=%s,status code=%d,use time=%v", string(body), resp.StatusCode, time.Since(startTime))

	return true
}

func getPhpURL(channel int) string {
	switch channel {
	case user_message_center_api.ChannelXNGService:
		return conf.C.PhpReceiveAddr["xngservice"]
	case user_message_center_api.ChannelXNGMiniApp:
		return conf.C.PhpReceiveAddr["xngminiapp"]
	case user_message_center_api.ChannelXNGSubscribe:
		return conf.C.PhpReceiveAddr["xngsubscribe"]
	}
	return ""
}
