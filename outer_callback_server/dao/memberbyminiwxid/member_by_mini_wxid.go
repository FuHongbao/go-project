package memberbyminiwxid

import (
	"errors"
	"fmt"
	mgo "xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/lib/xmongo"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/conf"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/utils"
)

//Dao xmongo split client defined
var Dao *xmongo.SplitClient

func dbFunc(v map[string]interface{}) (session *mgo.Session, dbName string, err error) {
	_id := v["_id"].(string)
	num := utils.OpenidToNum(_id)
	modNum := num % 4
	dbName = fmt.Sprintf("xng_user_%d", modNum)
	session = conf.DBS[dbName]
	if session == nil {
		err = errors.New("session nil")
		return
	}
	return
}

func colFunc(v map[string]interface{}) (colName string, err error) {
	_id := v["_id"].(string)
	num := utils.OpenidToNum(_id)
	modNum := num % 4096
	colName = fmt.Sprintf("member_by_mini_wxid_%d", modNum)
	return
}

func init() {
	var err error
	Dao, err = xmongo.NewSplitClient([]string{"_id"}, dbFunc, colFunc)
	if err != nil {
		xlog.Fatal("Create user Xmongo Client Failed: %v", err)
	}
}
