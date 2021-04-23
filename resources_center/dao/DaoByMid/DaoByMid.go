package DaoByMid

import (
	"fmt"
	"github.com/pkg/errors"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo"
	"xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo/bson"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/lib/xmongo"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao"
)

var DaoMid *xmongo.SplitClient

func init() {
	var err error
	DaoMid, err = xmongo.NewSplitClient([]string{"mid"}, DBFunc, ColFunc)
	if err != nil {
		xlog.Fatal("Create Xmongo Client Failed: %v", err)
		return
	}
}

func DBFunc(v map[string]interface{}) (*mgo.Session, string, error) {
	mid, ok := v["mid"].(int64)
	if !ok {
		xlog.Error("DBFunc get mid Err, v:%v", v)
		return nil, "", errors.New("DBFunc get mid error")
	}

	resMod := dao.GetResMod(mid)
	nodeMod := dao.GetNodeMod(resMod, dao.DbUserResource)
	spliteStr := fmt.Sprintf("%s_%d", dao.DbUserResource, nodeMod)
	mgoSession, ok := conf.DBS[spliteStr]
	if !ok {
		xlog.Error("get mid session error, mid:%v", v["mid"])
		return nil, spliteStr, errors.New("get mgo session err ")
	}
	return mgoSession, spliteStr, nil
}

func ColFunc(v map[string]interface{}) (string, error) {
	mid, ok := v["mid"].(int64)
	if !ok {
		xlog.Error("col func get mid error, v:%v", v)
		return "", errors.New("get mid error ")
	}
	resMod := dao.GetResMod(mid)
	spliteStr := fmt.Sprintf("%s_%d", dao.ColUserResByMid, resMod)
	return spliteStr, nil
}

func GetDocByMid(mid int64, qid int64) (doc *api.UserResourceDoc, err error) {
	splitVal := map[string]interface{}{"mid": mid}
	qry := bson.M{"mid": mid, "qid": qid}
	err = DaoMid.FindOne(splitVal, qry, &doc, nil)
	if err == mgo.ErrNotFound {
		err = nil
		return
	}
	if err != nil {
		doc = nil
		return
	}
	return
}

func UpdateUserResourceDoc(qry bson.M, upData bson.M, mid int64) (err error) {
	splitVal := map[string]interface{}{"mid": mid}
	err = DaoMid.Update(splitVal, qry, upData)
	if err != nil {
		return
	}
	return nil
}

func AddResourceDoc(mid int64, doc *api.UserResourceDoc) (err error) {
	splval := map[string]interface{}{"mid": mid}
	upt := time.Now().UnixNano() / 1e6
	var mt int64
	if doc.Ct == 0 {
		mt = upt
	}
	doc.Upt = upt
	doc.Mt = mt
	err = DaoMid.Insert(splval, doc)
	if err != nil {
		return
	}
	return
}
