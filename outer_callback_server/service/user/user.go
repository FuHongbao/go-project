package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/lib/xmongo"
	"xgit.xiaoniangao.cn/xngo/lib/xnet/xhttp"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/conf"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/dao/memberbyminiwxid"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/dao/memberbywxid"
	"xgit.xiaoniangao.cn/xngo/service/user_message_center_api"
)

//MemberByOpenidEntity 根据openid获取用户信息
type MemberByOpenidEntity struct {
	ID  string `json:"id" bson:"_id"`
	Mid int64  `json:"mid" bson:"mid"`
}

//WxOpenidToMid 暂时查数据库，需要改为调用户中心
func WxOpenidToMid(appid string, openid string) (mid int64, err error) {
	var dao *xmongo.SplitClient
	switch appid {
	case conf.C.Wxids["xngservice"]:
		dao = memberbywxid.Dao
	case conf.C.Wxids["xngminiapp"]:
		dao = memberbyminiwxid.Dao
	}

	if dao == nil {
		err = errors.New("appid not exist")
		return
	}

	ent := &MemberByOpenidEntity{}
	err = dao.FindId(map[string]interface{}{"_id": openid}, openid, ent)
	if err != nil {
		return
	}

	mid = ent.Mid

	return
}

//MidByOpenidReq defined
type MidByOpenidReq struct {
	Openid string `json:"openid"`
	Type   string `json:"type"`
}

//MidByOpenidRsp defined
type MidByOpenidRsp struct {
	Ret  int `json:"ret"`
	Data struct {
		Mid int64 `json:"mid"`
	} `json:"data"`
}

func wxidToType(wxid string) string {
	switch wxid {
	case conf.C.Wxids["xngservice"]:
		return "xng_mp"
	case conf.C.Wxids["xngminiapp"]:
		return "xng_miniapp"
	case conf.C.Wxids["xngsubscribe"]:
		return "xng_sub"
	case conf.C.Wxids["xngapp"]:
		return "xng_app"
	case conf.C.Wxids["xbdminiapp"]:
		return "xbd_miniapp"
	case conf.C.Wxids["gameidiomminiapp"]:
		return "game_idiom"
	case conf.C.Wxids["gameduetminiapp"]:
		return "dc_miniapp"
	}

	return ""
}

func httpPost(url string, req *MidByOpenidReq) (mid int64, err error) {
	var httpRsp *http.Response
	var client = xhttp.NewClient()
	if httpRsp, err = client.PostJson(url, req); err != nil {
		xlog.Error("get mid err,url=%s, err=%s", url, err.Error())
		return
	}
	defer httpRsp.Body.Close()

	var body []byte
	if body, err = ioutil.ReadAll(httpRsp.Body); err != nil {
		xlog.Error("read body fail,err=%s", err.Error())
		return
	}

	var midbyopenidRsp = &MidByOpenidRsp{}
	if err = json.Unmarshal(body, midbyopenidRsp); err != nil {
		xlog.Error("unmarshall body fail, err=%s", err.Error())
		return
	}

	if midbyopenidRsp.Ret != 1 {
		err = errors.New("ret not 1")
		xlog.Error("ret not 1, rsp=%v", midbyopenidRsp)
		return
	}

	mid = midbyopenidRsp.Data.Mid

	return
}

//GetMid 根据openid 和 appid 获取mid
func GetMid(wxid, openid string) (mid int64, err error) {
	var url, ipport, userCenterSrv string
	var req = &MidByOpenidReq{Openid: openid, Type: wxidToType(wxid)} //需要在用户中心添加对应配置

	if userCenterSrv = conf.C.Addrs.UserCenter; userCenterSrv == "" {
		err = errors.New("user center addr empty")
		return
	}

	if ipport, err = lib.NameWrap(userCenterSrv); err != nil {
		return
	}
	xlog.Debug("ipport=%s", ipport)
	url = fmt.Sprintf("http://%s/user/get_mid_by_openid", ipport)

	mid, err = httpPost(url, req)
	return
}

func GetMidByChannel(wxid, openid, msgType string, channel int) (mid int64, err error) {
	if msgType == "event" && (channel == user_message_center_api.ChannelXNGService || channel == user_message_center_api.ChannelXNGSubscribe) {
		return
	}
	mid, err = GetMid(wxid, openid)
	return
}
