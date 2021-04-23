package DaoByTag

import (
	"fmt"
	"github.com/pkg/errors"
	mgo "xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo"
	"xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo/bson"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/lib/xmongo"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao"
)

var DaoTag *xmongo.SplitClient

func init() {
	var err error
	DaoTag, err = xmongo.NewSplitClient([]string{"qetag"}, DBFunc, ColFunc)
	if err != nil {
		xlog.Fatal("Create Xmongo Client Failed: %v", err)
		return
	}
}

func DBFunc(v map[string]interface{}) (*mgo.Session, string, error) {
	qetag, ok := v["qetag"].(string)
	if !ok {
		xlog.Error("DBFunc get qetag  Err, v:%v", v)
		return nil, "", errors.New("get qetag Err ")
	}
	keyStr := dao.GetKeyStr(qetag)
	mediaId := dao.GetMediaId(keyStr)
	resMod := dao.GetResMod(mediaId)
	nodeMod := dao.GetNodeMod(resMod, dao.DbXngResource)
	spliteStr := fmt.Sprintf("%s_%d", dao.DbXngResource, nodeMod)
	mgoSession, ok := conf.DBS[spliteStr]
	if !ok {
		xlog.Error("get qetag session err, qetag:%v", v["qetag"])
		return nil, spliteStr, errors.New("get mgosession error")
	}
	return mgoSession, spliteStr, nil
}

func ColFunc(v map[string]interface{}) (string, error) {
	keyStr := dao.GetKeyStr(v["qetag"].(string))
	mediaId := dao.GetMediaId(keyStr)
	resMod := dao.GetResMod(mediaId)
	spliteStr := fmt.Sprintf("%s_%d", dao.ColCommonFileByTag, resMod)
	return spliteStr, nil
}

//利用qetag获取qid
func GetDocByTag(qeTag string) (doc *api.XngTagInfoDoc, err error) {
	splitVal := map[string]interface{}{"qetag": qeTag}
	qry := bson.M{"_id": qeTag}
	err = DaoTag.FindOne(splitVal, qry, &doc, nil)
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

func AddXngTagDoc(qetag string, qid int64) error {
	splval := map[string]interface{}{"qetag": qetag}
	//doc := api.XngTagInfoDoc{
	//	QeTag: qetag,
	//	Qid:   qid,
	//}
	q := bson.M{
		"_id": qetag,
	}
	up := bson.M{
		"$set": bson.M{"qid": qid},
	}
	_, err := DaoTag.Upsert(splval, q, up)
	if err != nil {
		return err
	}
	return nil
}
