package dao_by_qid

import ()

/*
var DaoQid *xmongo.SplitClient

func init() {
	var err error
	DaoQid, err = xmongo.NewSplitClient([]string{"qid"}, DBFunc, ColFunc)
	if err != nil {
		xlog.Fatal("Create Xmongo Client Failed: %v", err)
		return
	}
}

func DBFunc(v map[string]interface{}) (*mgo.Session, string, error) {
	qid, ok := v["qid"].(int64)
	if !ok {
		xlog.Error("DBFunc get qid Err, v:%v", v)
		return nil, "", errors.New("get qid Err ")
	}

	resMod := dao.GetResMod(qid)
	nodeMod := dao.GetNodeMod(resMod, dao.DbXngResource)
	spliteStr := fmt.Sprintf("%s_%d", dao.DbXngResource, nodeMod)
	mgoSession, ok := conf.DBS[spliteStr]
	if !ok {
		xlog.Error("get qid session err, qid:%v", v["qid"])
		return nil, spliteStr, errors.New("get mgosession err ")
	}
	return mgoSession, spliteStr, nil
}

func ColFunc(v map[string]interface{}) (string, error) {
	qid, ok := v["qid"].(int64)
	if !ok {
		xlog.Error("COLFunc get qid Err, v:%v", v)
		return "", errors.New("get qid Err ")
	}
	resMod := dao.GetResMod(qid)
	spliteStr := fmt.Sprintf("%s_%d", dao.ColCommonFileByQid, resMod)
	return spliteStr, nil
}

//更新资源信息
func UpdateResourceDoc(qid int64, qry bson.M, updata bson.M) error {
	splval := map[string]interface{}{"qid": qid}
	err := DaoQid.Update(splval, qry, updata)
	if err != nil {
		return err
	}
	return nil
}
func GetDocByQid(qid int64) (doc *api.XngResourceInfoDoc, err error) {
	splitVal := map[string]interface{}{"qid": qid}
	qry := bson.M{"_id": qid}
	err = DaoQid.FindOne(splitVal, qry, &doc, nil)
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
*/
