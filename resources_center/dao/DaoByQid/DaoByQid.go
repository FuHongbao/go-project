package DaoByQid

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"strings"
	mgo "xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo"
	"xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo/bson"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/lib/xmongo"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao"
)

var DaoQid *xmongo.SplitClient

//资源库文档格式（兼容库内字段类型不同问题）
type XngResourceDocForCompat struct {
	ResId     int64       `json:"_id" bson:"_id"`
	Type      int         `json:"ty" bson:"ty,omitempty"`
	Size      int64       `json:"size" bson:"size,omitempty"`
	QeTag     string      `json:"qetag" bson:"qetag,omitempty"`
	Upt       int64       `json:"upt" bson:"upt,omitempty"`
	Mt        int64       `json:"mt" bson:"mt,omitempty"`
	Ct        int64       `json:"ct,omitempty" bson:"ct,omitempty"`
	Src       string      `json:"src" bson:"src,omitempty"`
	Fmt       string      `json:"fmt" bson:"fmt,omitempty"`
	Ort       int         `json:"ort" bson:"ort,omitempty"`
	W         interface{} `json:"w" bson:"w,omitempty"`
	H         interface{} `json:"h" bson:"h,omitempty"`
	Ref       int         `json:"ref" bson:"ref,omitempty"`
	Du        float64     `json:"du,omitempty" bson:"du,omitempty"`
	Cover     int64       `json:"cover,omitempty" bson:"cover,omitempty"`
	CoverTp   string      `json:"covertp,omitempty" bson:"covertp,omitempty"`
	Code      string      `json:"code,omitempty" bson:"code,omitempty"`
	TransCode *int        `json:"trans,omitempty" bson:"trans"`
}

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

//利用qid获取资源文档信息
func GetDocByQid(qid int64) (doc *api.XngResourceInfoDoc, err error) {
	splitVal := map[string]interface{}{"qid": qid}
	qry := bson.M{"_id": qid}
	compatDoc := &XngResourceDocForCompat{}
	err = DaoQid.FindOne(splitVal, qry, &compatDoc, nil)
	if err == mgo.ErrNotFound {
		err = nil
		return
	}
	if err != nil {
		doc = nil
		return
	}
	doc = &api.XngResourceInfoDoc{
		ResId:     compatDoc.ResId,
		Type:      compatDoc.Type,
		Size:      compatDoc.Size,
		QeTag:     compatDoc.QeTag,
		Upt:       compatDoc.Upt,
		Mt:        compatDoc.Mt,
		Ct:        compatDoc.Ct,
		Src:       compatDoc.Src,
		Fmt:       compatDoc.Fmt,
		Ort:       compatDoc.Ort,
		Ref:       compatDoc.Ref,
		Du:        compatDoc.Du,
		Cover:     compatDoc.Cover,
		CoverTp:   compatDoc.CoverTp,
		Code:      compatDoc.Code,
		TransCode: compatDoc.TransCode,
	}
	if compatDoc.W == nil || compatDoc.H == nil { //音乐，文本没有w和h字段
		return
	}
	if doc.Code != "" && strings.Contains(doc.Code, "avc") {
		doc.Code = "h264"
	} else if doc.Code != "" && strings.Contains(doc.Code, "hevc") {
		doc.Code = "h265"
	}
	doc.W = cast.ToInt(compatDoc.W)
	doc.H = cast.ToInt(compatDoc.H)
	//switch compatDoc.W.(type) {
	//case string:
	//	width, errIgnore := strconv.Atoi(compatDoc.W.(string))
	//	if errIgnore != nil {
	//		err = errIgnore
	//		return
	//	}
	//	doc.W = width
	//case int:
	//	doc.W = compatDoc.W.(int)
	//default:
	//	err = errors.New(fmt.Sprintf("GetDocByQid.W type:[%v] is unsupport", reflect.TypeOf(compatDoc.W)))
	//	return
	//}
	//switch compatDoc.H.(type) {
	//case string:
	//	height, errIgnore := strconv.Atoi(compatDoc.H.(string))
	//	if errIgnore != nil {
	//		err = errIgnore
	//		return
	//	}
	//	doc.H = height
	//case int:
	//	doc.H = compatDoc.H.(int)
	//default:
	//	err = errors.New(fmt.Sprintf("GetDocByQid.H type:[%v] is unsupport", reflect.TypeOf(compatDoc.H)))
	//	return
	//}
	//err = DaoQid.FindOne(splitVal, qry, &doc, nil)
	//if err == mgo.ErrNotFound {
	//	err = nil
	//	return
	//}
	//if err != nil {
	//	doc = nil
	//	return
	//}
	return
}

func UpdateResourceRef(qid int64) (err error) {
	splval := map[string]interface{}{"qid": qid}
	qry := bson.M{"_id": qid}
	updata := bson.M{"$inc": bson.M{"ref": 1}}
	err = DaoQid.Update(splval, qry, updata)
	if err != nil {
		return
	}
	return nil
}

//更新资源信息
func UpdateResourceDoc(qid int64, qry bson.M, updata bson.M) error {
	splval := map[string]interface{}{"qid": qid}
	//FIXME：update会把整个文档会直接删掉原来的数据，使用updata里面的数据，要想对某个或某些字段赋值，要用$set。注意：所有的update和upsert都检查一遍
	err := DaoQid.Update(splval, qry, updata)
	if err != nil {
		return err
	}
	return nil
}

//添加资源信息
func InsertResourceDoc(qid int64, qDoc *api.XngResourceInfoDoc) error {
	splitVal := map[string]interface{}{"qid": qid}
	err := DaoQid.Insert(splitVal, qDoc)
	if err != nil {
		return err
	}
	return nil
}

//更新资源信息
func UpInsertResourceDoc(qid int64, update map[string]interface{}) error {
	splval := map[string]interface{}{"qid": qid}
	q := bson.M{
		"_id": qid,
	}
	up := bson.M{
		"$set": update,
	}
	//upByte, err := bson.Marshal(doc)
	//if err != nil {
	//	return err
	//}
	//var update bson.M
	//err = bson.Unmarshal(upByte, &update)
	//if err != nil {
	//	return err
	//}
	//up := bson.D{{"$set", update}}
	_, err := DaoQid.Upsert(splval, q, up)
	if err != nil {
		return err
	}
	return nil
}
