package wxmediadao

import (
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/lib/xmongo"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/conf"
)

//WXMedia defined
type WXMedia struct {
	ID         int64  `json:"id" bson:"_id"`
	FUser      string `json:"fuser" bson:"fuser"`
	TUser      string `json:"tuser" bson:"tuser"`
	Type       string `json:"type" bson:"type"`
	Src        int    `json:"src" bson:"src"`
	Wxct       int64  `json:"wxct" bson:"wxct"`
	Purl       string `json:"purl" bson:"purl"`
	Meid       string `json:"meid" bson:"meid"`
	Msid       int64  `json:"msid" bson:"msid"`
	Mid        int64  `json:"mid" bson:"mid"`
	Tmid       int64  `json:"tmid" bson:"tmid"`
	Ct         int64  `json:"ct" bson:"ct"`
	Status     int    `json:"status" bson:"status"`
	Fmt        string `json:"fmt" bson:"fmt"`
	IsCallBack bool   `json:"is_call_back"`
	MsgID      string `json:"msg_id"`
}

//Dao xmongo client defined
var Dao *xmongo.Client

func init() {
	var err error
	Dao, err = xmongo.NewClient(conf.DBS["wx"], "xng_wx", "msg")
	if err != nil {
		xlog.Fatal("Create Msg Xmongo Client Failed: %v", err)
	}
}
