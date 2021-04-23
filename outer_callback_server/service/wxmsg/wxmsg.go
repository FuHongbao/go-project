package wxmsg

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo/bson"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/conf"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/dao/wxmediadao"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/service/ids"
	userService "xgit.xiaoniangao.cn/xngo/service/outer_callback_server/service/user"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/service/wxmedia"
	"xgit.xiaoniangao.cn/xngo/service/user_message_center_api"
)

//WxRawMsg 微信各种消息类型所包含的字段
type WxRawMsg struct {
	//通用字段
	ToUserName   string `xml:"ToUserName" json:"ToUserName"`
	FromUserName string `xml:"FromUserName" json:"FromUserName"`
	CreateTime   int64  `xml:"CreateTime" json:"CreateTime"`
	MsgType      string `xml:"MsgType" json:"MsgType"`
	MsgID        int64  `xml:"MsgId" json:"MsgId"`

	//文本消息特有
	Content string `xml:"Content" json:"Content,omitempty"`

	//图片消息特有
	PicURL string `xml:"PicUrl" json:"PicUrl,omitempty"`

	//图片、视频、卡片消息特有
	MediaID string `xml:"MediaId" json:"MediaId,omitempty"`

	//视频消息特有
	ThumbMediaID string `xml:"ThumbMediaId" json:"ThumbMediaId,omitempty"`

	//语音消息特有
	Format string `xml:"Format" json:"Format,omitempty"`

	//地理位置特有
	LocationX string `xml:"LocationX" json:"LocationX,omitempty"`
	LocationY string `xml:"LocationY" json:"LocationY,omitempty"`
	Scale     string `xml:"Scale" json:"Scale,omitempty"`
	Label     string `xml:"Label" json:"Label,omitempty"`

	//链接消息特有
	Title       string `xml:"Title" json:"Title,omitempty"`
	Description string `xml:"Description" json:"Description,omitempty"`
	URL         string `xml:"Url" json:"Url,omitempty"`

	//卡片消息特有
	PagePath string `xml:"PagePath" json:"PagePath,omitempty"`
	ThumbURL string `xml:"ThumbUrl" json:"ThumbUrl,omitempty"`

	//事件消息
	Event    string `xml:"Event" json:"Event,omitempty"`
	EventKey string `xml:"EventKey" json:"EventKey,omitempty"`
	Ticket   string `xml:"Ticket" json:"Ticket,omitempty"`

	Latitude  string `xml:"Latitude" json:"Latitude,omitempty"`
	Longitude string `xml:"Longitude" json:"Longitude,omitempty"`
	Precision string `xml:"Precision" json:"Precision,omitempty"`
}

//WxQuery defined
type WxQuery struct {
	Signature string `form:"signature"`
	Timestamp string `form:"timestamp"`
	Nonce     string `form:"nonce"`
	Echostr   string `form:"echostr"`
}

//CheckWxSignature 签名检查
func CheckWxSignature(wxquery *WxQuery, token string) bool {
	if token == "" {
		xlog.Error("token is empty, maybe not configed")
		return false
	}
	if wxquery.Signature == "" || wxquery.Timestamp == "" || wxquery.Nonce == "" {
		xlog.Error("param error")
		return false
	}

	si := []string{token, wxquery.Timestamp, wxquery.Nonce}
	sort.Strings(si)              //字典序排序
	str := strings.Join(si, "")   //组合字符串
	s := sha1.New()               //返回一个新的使用SHA1校验的hash.Hash接口
	_, _ = io.WriteString(s, str) //WriteString函数将字符串数组str中的内容写入到s中
	genSignature := fmt.Sprintf("%x", s.Sum(nil))

	xlog.Debug("wxquery=%v,genSignatrue=%s", *wxquery, genSignature)
	return genSignature == wxquery.Signature
}

//WxidToChannel 通过账号原始id获取channel
func WxidToChannel(wxid string) int {
	if wxid == "" {
		xlog.Error("wxid is empty")
		return user_message_center_api.ChannelUndefine
	}

	switch wxid {
	case conf.C.Wxids["xngservice"]:
		return user_message_center_api.ChannelXNGService
	case conf.C.Wxids["xngminiapp"]:
		return user_message_center_api.ChannelXNGMiniApp
	case conf.C.Wxids["xngsubscribe"]:
		return user_message_center_api.ChannelXNGSubscribe
	case conf.C.Wxids["xngapp"]:
		return user_message_center_api.ChannelXNGAndroid
	case conf.C.Wxids["xbdservice"]:
		return user_message_center_api.ChannelXBDService
	case conf.C.Wxids["xbdsubscribe"]:
		return user_message_center_api.ChannelXBDSubscribe
	case conf.C.Wxids["xbdminiapp"]:
		return user_message_center_api.ChannelXBDMiniApp
	case conf.C.Wxids["tiaservice"]:
		return user_message_center_api.ChannelTIAService
	case conf.C.Wxids["gameidiomminiapp"]:
		return user_message_center_api.ChannelGameIdiomMiniApp
	case conf.C.Wxids["gameduetminiapp"]: //对唱小游戏
		return user_message_center_api.ChannelGameDuetMiniApp
	}

	xlog.Error("no matched wxid, maybe wxid not configured")
	return user_message_center_api.ChannelUndefine
}

//UploadMediaTask defined
type UploadMediaTask struct {
	ID    int64 `json:"id"`
	Retry int   `json:"retry"`
	Data  struct {
		Fuser      string `json:"fuser"`
		WxMsgID    int64  `json:"wx_msg_id"`
		PicURL     string `json:"pic_url"`
		Mid        int64  `json:"mid"`
		MsgID      string `json:"msg_id"`
		Src        int    `json:"src"`
		IsCallBack bool   `json:"is_call_back"`
	} `json:"data"`
}

//UploadVoiceTask defined
type UploadVoiceTask struct {
	ID    int64               `json:"id"`
	Retry int                 `json:"retry"`
	Data  *wxmediadao.WXMedia `json:"data"`
}

//AddUploadWxImageTask 增加下载图片任务
func AddUploadWxImageTask(wxRawMsg *WxRawMsg, req *user_message_center_api.RevReq) {
	redisPool := conf.RDS["weixin"]
	conn := redisPool.Get()
	defer conn.Close()
	taskID, err := redis.Int64(conn.Do("incr", "wx_img_taskid"))
	if err != nil || taskID == 0 {
		xlog.Error("incr redis fail")
		return
	}
	task := UploadMediaTask{}

	id, err := ids.GetMsgID()
	if err != nil {
		return
	}
	wxmsg := &wxmediadao.WXMedia{}
	wxmsg.ID = id
	wxmsg.Mid = req.Mid
	wxmsg.FUser = wxRawMsg.FromUserName
	wxmsg.TUser = wxRawMsg.ToUserName
	wxmsg.Type = wxRawMsg.MsgType
	wxmsg.Src = req.Channel
	wxmsg.Wxct = wxRawMsg.CreateTime
	wxmsg.Purl = wxRawMsg.PicURL
	wxmsg.Meid = wxRawMsg.MediaID
	wxmsg.Msid = wxRawMsg.MsgID
	wxmsg.Ct = time.Now().Unix()
	wxmsg.Tmid = 0
	wxmsg.Status = 2

	err = wxmedia.Add(wxmsg)
	if err != nil {
		return
	}
	task.ID = taskID
	task.Retry = 0
	task.Data.Fuser = wxRawMsg.FromUserName
	task.Data.WxMsgID = id
	task.Data.PicURL = wxRawMsg.PicURL
	task.Data.Mid = req.Mid
	task.Data.MsgID = strconv.FormatInt(wxRawMsg.MsgID, 10)
	task.Data.Src = req.Channel
	task.Data.IsCallBack = true

	req.MsgId = strconv.FormatInt(wxRawMsg.MsgID, 10)

	jsonData, err := json.Marshal(task)
	if err != nil {
		xlog.Error("task marshal fail,task %v,err %v", task, err)
		return
	}
	data, err := redis.Int64(conn.Do("lpush", "wx_img_queue_new", jsonData))
	if err != nil || data == 0 {
		xlog.Error("push to redis fail")
		return
	}
	xlog.Debug("push to redis,task=%v", task)
	return
}

//AddUploadWxVoiceTask 增加下载视频任务
func AddUploadWxVoiceTask(wxRawMsg *WxRawMsg, req *user_message_center_api.RevReq) {
	redisPool := conf.RDS["weixin"]
	conn := redisPool.Get()
	defer conn.Close()

	taskID, err := redis.Int64(conn.Do("incr", "wx_voice_taskid"))
	if err != nil || taskID == 0 {
		xlog.Error("incr redis fail")
		return
	}
	xlog.Debug("taskId=%d", taskID)

	task := &UploadVoiceTask{}

	id, err := ids.GetMsgID()
	if err != nil {
		return
	}

	wxmsg := &wxmediadao.WXMedia{}
	wxmsg.ID = id
	wxmsg.Mid = req.Mid
	wxmsg.Type = wxRawMsg.MsgType
	wxmsg.Src = req.Channel
	wxmsg.Meid = wxRawMsg.MediaID
	wxmsg.Msid = wxRawMsg.MsgID
	wxmsg.Ct = time.Now().Unix()
	wxmsg.Fmt = wxRawMsg.Format
	wxmsg.IsCallBack = true
	wxmsg.MsgID = strconv.FormatInt(wxRawMsg.MsgID, 10)

	err = wxmedia.Add(wxmsg)
	if err != nil {
		return
	}

	task.ID = taskID
	task.Retry = 0
	task.Data = wxmsg

	req.MsgId = strconv.FormatInt(wxRawMsg.MsgID, 10)

	jsonData, err := json.Marshal(task)
	if err != nil {
		xlog.Error("task marshal fail,task %v,err %v", task, err)
	}
	data, err := redis.Int64(conn.Do("lpush", "wx_voice_queue_new", jsonData))
	xlog.Info("push voice msg to queue: %v", *task)
	if err != nil || data == 0 {
		xlog.Error("push to redis fail")
		return
	}
	xlog.Debug("push to redis,task=%v,data=%d", task, data)
	return
}

//NormalizeWxRawMsg 组装消息
func NormalizeWxRawMsg(wxRawMsg *WxRawMsg) (*user_message_center_api.RevReq, error) {
	var (
		mid    int64
		err    error
		xngMsg = &user_message_center_api.RevReq{}
	)
	xngMsg.Channel = WxidToChannel(wxRawMsg.ToUserName)
	if mid, err = userService.GetMidByChannel(wxRawMsg.ToUserName, wxRawMsg.FromUserName, wxRawMsg.MsgType, xngMsg.Channel); err != nil {
		xlog.Error("openid to mid fail,err=%s", err)
		return nil, err
	}
	xlog.Debug("NormalizeWxRawMsg.GetMidByChannel mid:[%v]", mid)
	xngMsg.Mid = mid
	xngMsg.MsgId = bson.NewObjectId().Hex()
	xngMsg.Ct = int64(wxRawMsg.CreateTime) * 1000

	switch wxRawMsg.MsgType {
	case "text":
		xngMsg.Type = user_message_center_api.MsgTypeText
		xngMsg.Body = map[string]interface{}{"txt": wxRawMsg.Content}
	case "image":
		xngMsg.Type = user_message_center_api.MsgTypeImg
		xngMsg.Body = map[string]interface{}{"pic_url": wxRawMsg.PicURL, "media_id": wxRawMsg.MediaID}
		AddUploadWxImageTask(wxRawMsg, xngMsg) //添加到下载队列
	case "voice":
		xngMsg.Type = user_message_center_api.MsgTypeVoice
		xngMsg.Body = map[string]interface{}{"media_id": wxRawMsg.MediaID, "format": wxRawMsg.Format}
		AddUploadWxVoiceTask(wxRawMsg, xngMsg)
	case "video":
		xngMsg.Type = user_message_center_api.MsgTypeVideo
		xngMsg.Body = map[string]interface{}{"media_id": wxRawMsg.MediaID, "thumb_media_id": wxRawMsg.ThumbMediaID}
		//AddUploadWxMediaTask(wxRawMsg.MsgId, wxRawMsg.FromUserName, wxRawMsg.PicUrl, xngMsg) //添加到下载队列
	case "shortvideo":
		xngMsg.Type = user_message_center_api.MsgTypeShortVideo
		xngMsg.Body = map[string]interface{}{"media_id": wxRawMsg.MediaID, "thumb_media_id": wxRawMsg.ThumbMediaID}
	case "location":
		xngMsg.Type = user_message_center_api.MsgTypeLocation
		xngMsg.Body = map[string]interface{}{"location_x": wxRawMsg.LocationX, "location_y": wxRawMsg.LocationY, "scale": wxRawMsg.Scale, "label": wxRawMsg.Label}
	case "link":
		xngMsg.Type = user_message_center_api.MsgTypeLink
		xngMsg.Body = map[string]interface{}{"title": wxRawMsg.Title, "description": wxRawMsg.Description, "url": wxRawMsg.URL}
	case "miniprogrampage":
		xngMsg.Type = user_message_center_api.MsgTypeMiniAppCard
		xngMsg.Body = map[string]interface{}{"title": wxRawMsg.Title, "pagepath": wxRawMsg.PagePath, "thumb_media_id": wxRawMsg.ThumbMediaID}
		// 只有缩略图，是否要下载？？
		//AddUploadWxMediaTask(wxRawMsg.MsgId, wxRawMsg.FromUserName, wxRawMsg.ThumbUrl, xngMsg) //添加到下载队列

	case "event":
		xngMsg.Type = user_message_center_api.MsgTypeEvent
		xngMsg.Body = map[string]interface{}{"event": wxRawMsg.Event, "event_key": wxRawMsg.EventKey, "ticket": wxRawMsg.Ticket, "openid": wxRawMsg.FromUserName}
	}

	return xngMsg, nil
}
