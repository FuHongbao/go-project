package videoService

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/mts"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo/bson"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/ids_api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByMid"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByQid"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByTag"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/redis/videoRedis"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/sts"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils"
)

const (
	TransStatusOldOver = 3
	TransStatusOver    = 2
	TransStatusSubmit  = 1
	TransStatusNot     = 0

	SubmitStatusSuccess  = 1
	SubmitStatusFail     = 0
	MakeAlbumFailErrorNo = -5
	TransListBaseStr     = "trans:list:"
	UserUpdateBaseStr    = "user:list:"
	HandleInfoBaseStr    = "info:list:"
)

const (
	UploadStatusNot     = 0
	UploadStatusTrying  = 1
	UploadStatusSuccess = 2
	UploadStatusCancel  = 3
	UploadStatusFail    = 4
)

var ResStatusNot = 0
var ResStatusTrans = 1
var ResStatusMade = 2

//QuerySnapTryTime = 240

type StsData struct {
	Endpoint      string `json:"endpoint"`
	EndpointInter string `json:"endpoint_internal"`
	InputBucket   string `json:"input_bucket"`
	OutputBucket  string `json:"output_bucket"`
	AccessKey     string `json:"accessKey"`
	SecretKey     string `json:"secretKey"`
	SecurityToken string `json:"securityToken"`
}

type StsInfo struct {
	Status int     `json:"ret"`
	Data   StsData `json:"data"`
}

type StsInfoForUserData struct {
	Endpoint      string `json:"endpoint"`
	EndpointInter string `json:"endpoint_internal"`
	Bucket        string `json:"bucket"`
	AccessKey     string `json:"accessKey"`
	SecretKey     string `json:"secretKey"`
	SecurityToken string `json:"securityToken"`
}
type StsInfoForUser struct {
	Status int                `json:"ret"`
	Data   StsInfoForUserData `json:"data"`
}

type CallbackParam struct {
	CallbackUrl      string `json:"callbackUrl"`
	CallbackBody     string `json:"callbackBody"`
	CallbackBodyType string `json:"callbackBodyType"`
}

type ConfigStruct struct {
	Expiration string     `json:"expiration"`
	Conditions [][]string `json:"conditions"`
}

var GlobalStsInfoForMts *StsInfo

var mutex sync.Mutex

func init() {
	go func() {
		GlobalStsInfoForMts, _ = getStsForMtsFromRemote()
		for {
			var err error
			select { //每半小时更新一次
			case <-time.After(time.Minute * 20):
				for i := 0; i < api.ReTryTimes; i++ {
					mutex.Lock()
					GlobalStsInfoForMts, err = getStsForMtsFromRemote()
					mutex.Unlock()
					if err == nil && GlobalStsInfoForMts != nil {
						break
					}
					xlog.Error("init fail to get sts info for mts, err:%v", err)
				}
			}
		}
	}()
}

type ResourceModel interface {
	GetResourceInfo(req *api.MediaInfoReq) (*api.MediaInfoResp, error)
	TransCodeResource(*api.TransCodeReq) (int, error)
	HandleNewResource(req *api.SetUploadStatusReq) error
}

type MediaVideo struct {
	resType int
}

func (v *MediaVideo) HandleNewResource(req *api.SetUploadStatusReq) (err error) {
	stsInfo, err := GetStsForMts()
	if err != nil || stsInfo == nil {
		xlog.Error("HandleNewResource get sts information err:%v, req:%v", err, req)
		return
	}
	//endPoint := stsInfo.Data.EndpointInter
	//if conf.Env != lib.PROD {
	//	endPoint = stsInfo.Data.Endpoint
	//}
	endPoint := stsInfo.Data.Endpoint
	regionID := endPoint[4 : len(endPoint)-13]
	client, err := mts.NewClientWithStsToken(regionID, stsInfo.Data.AccessKey, stsInfo.Data.SecretKey, stsInfo.Data.SecurityToken)
	if err != nil {
		xlog.Error("HandleNewResource create new client with stsToken error:%v, req:%v", err, req)
		return
	}
	if conf.Env == lib.PROD {
		client.Domain = api.MtsVpcDomain
	}
	ch := make(chan int64, 1)
	go GetDistributedId(ch)

	uploadBucket := stsInfo.Data.InputBucket
	loc := endPoint[:len(endPoint)-13]
	fileName := strconv.FormatInt(req.Qid, 10)
	doc, err := GetVideoInfo(client, uploadBucket, loc, fileName)
	if err != nil || doc == nil {
		xlog.Error("HandleNewResource get video information error:%v, req:%v", err, req)
		return
	}
	//todo: 截图类型先写死，以后再看
	snapType := "img"
	snapId := <-ch
	if snapId <= 0 {
		err = errors.New("HandleNewResource get snapId err")
		return
	}

	go func() {
		errIgnore := SubmitSnapShotFrame(stsInfo, req.Qid, snapId, doc.Du)
		if errIgnore != nil {
			xlog.Error("failed to submit resource snap shot, snapId:%v, snapType:%v, err:%v", snapId, snapType, errIgnore)
		}

		ch <- req.Qid
	}()
	//doc.ResId = req.Qid
	doc.QeTag = req.QeTag
	doc.Cover = snapId
	doc.CoverTp = snapType
	err = SaveVideoInfoTemp(req.Qid, doc)
	//等待截图完成
	<-ch
	if err != nil {
		return
	}
	return
}

func AddSnapInfoToDB(snapQid int64, stsInfo *StsInfo) error {
	//获取自增ID作为截图的资源ID
	//ch := make(chan int64, 1)
	//go GetDistributedId(ch)
	//endPoint := stsInfo.Data.EndpointInter
	//if conf.Env != lib.PROD {
	//	endPoint = stsInfo.Data.Endpoint
	//}
	endPoint := stsInfo.Data.Endpoint
	regionID := endPoint[4 : len(endPoint)-13]
	client, err := mts.NewClientWithStsToken(regionID, stsInfo.Data.AccessKey, stsInfo.Data.SecretKey, stsInfo.Data.SecurityToken)
	if err != nil || client == nil {
		xlog.Error("create new client with stsToken error:%v", err)
		return err
	}
	if conf.Env == lib.PROD {
		client.Domain = api.MtsVpcDomain
	}
	location := endPoint[:len(endPoint)-13]
	bucket := conf.C.Bucket.Resource
	//now := time.Now()
	request := mts.CreateSubmitMediaInfoJobRequest()
	request.Scheme = "https"
	request.Async = requests.NewBoolean(false)
	request.Input = fmt.Sprintf("{\"Bucket\":\"%s\",\"Location\":\"%s\",\"Object\":\"%d\"}", bucket, location, snapQid)
	i := 0
	var response *mts.SubmitMediaInfoJobResponse
	var errIgnore error
	for i < api.ReTryTimes {
		response, errIgnore = client.SubmitMediaInfoJob(request)
		if errIgnore != nil || response == nil {
			xlog.Error("AddSnapInfoToDB submit media info job error, request: %v, response: %v, error: %v", request, response, err)
			continue
		}
		if response.MediaInfoJob.State == "Success" {
			break
		}
		i++
	}
	if errIgnore != nil || response == nil || response.MediaInfoJob.State != "Success" {
		//conf.Logger.Error("get snap info err or state not success", "err", errIgnore, "respone", response)
		return errors.New(fmt.Sprintf("AddSnapInfoToDB get snap info err or response is nil or state not success, err:%v, response:%v", errIgnore, response))
	}
	//conf.Logger.Info("Test over snap info", "time:", time.Since(now))
	fsize, err := strconv.ParseInt(response.MediaInfoJob.Properties.FileSize, 10, 64)
	if err != nil {
		xlog.Error("AddSnapInfoToDB failed to change string to int, resp:%v", response.MediaInfoJob.Properties)
		return err
	}
	width, err := strconv.Atoi(response.MediaInfoJob.Properties.Width)
	if err != nil {
		return err
	}
	height, err := strconv.Atoi(response.MediaInfoJob.Properties.Height)
	if err != nil {
		return err
	}
	upt := time.Now().UnixNano() / 1e6
	mt := upt
	qdoc := &api.XngResourceInfoDoc{
		ResId: snapQid,
		Type:  api.ResourceTypeImg,
		Size:  fsize,
		Upt:   upt,
		Mt:    mt,
		Ct:    0,
		Src:   api.UploadFromWXMiniApp,
		Fmt:   api.SnapShotType,
		Ort:   1,
		W:     width,
		H:     height,
		Ref:   1,
	}
	err = DaoByQid.InsertResourceDoc(snapQid, qdoc)
	if err != nil {
		//conf.Logger.Error("add new resource doc error, qid: %v, doc: %v, error: %v", snapQid, qdoc, err)
		return err
	}

	return nil
}

func GetNewResInfoFromRedis(req *api.MediaInfoReq) (doc *api.XngResourceInfoDoc, err error) {
	keyName := GetHandleInfoKey(req.Qid)
	doc, err = videoRedis.GetNewResInfoForHandle(keyName)
	if err != nil {
		return
	}
	errIgnore := videoRedis.DelNewResInfoForHandle(keyName)
	if errIgnore != nil {
		xlog.Error("failed to delete video resource in redis, req:%v, err:%v", req, errIgnore)
	}
	return
}

// 到这里就已经上传完成了
func (v *MediaVideo) GetResourceInfo(req *api.MediaInfoReq) (resp *api.MediaInfoResp, err error) {
	//尝试使用qetag获取资源doc
	qDoc, err := GetResDocByQeTag(req.QeTag)
	if err != nil {
		xlog.Error("GetResourceInfo get resource doc by qeTag err, req:%v, err:%v", req, err)
		return
	}
	start := time.Now()
	stsInfo, err := GetStsForMts()
	if err != nil || stsInfo == nil {
		xlog.Error("GetResourceInfo get sts information error, req:%v, err:%v", req, err)
		return
	}
	xlog.Debug("Test over sts info, time:%v", time.Since(start))
	var resID int64
	ch := make(chan struct{}, 1)
	// req.qid != 0 用于判断是否为重复资源，重复资源前端调用qid会传入0， qdoc存储的qid与请求传入的qid进行不相等的判断用于两人同时进行上传时的处理
	if qDoc == nil || (req.Qid != 0 && qDoc.ResId != req.Qid) { //不存在资源信息，提交作业及截图并存储信息
		if req.Qid <= 0 {
			err = errors.New("GetResourceInfo get resource info failed, parameter qid or key error")
			return
		}
		start = time.Now()
		qDoc, err = GetNewResInfoFromRedis(req)
		if err != nil || qDoc == nil {
			xlog.Error("GetResourceInfo get new resource information error, req:%v, err:%v", req, err)
			return
		}
		go func() {
			errIgnore := AddSnapInfoToDB(qDoc.Cover, stsInfo)
			ch <- struct{}{}
			if errIgnore != nil {
				xlog.Error("failed to add snapInfo to DB, qdoc:%v, req:%v, err:%v", qDoc, req, errIgnore)
				return
			}
			xlog.Debug("succ add snap info to db, req:%v", req)
		}()

		//conf.Logger.Debug("Test get new resource doc", "time:", time.Since(start))
		start = time.Now()
		resID, err = AddNewResource(qDoc, req.Mid, req.QeTag)
		if err != nil {
			xlog.Error("add new resource information to DB error, req:%v, qdoc:%v, err:%v", req, qDoc, err)
			return
		}
		//conf.Logger.Debug("Test over add doc", "time:", time.Since(start))
	} else {
		resID, err = UpdateResourceRecord(qDoc, qDoc.ResId, req.Mid)
		ch <- struct{}{}
		if err != nil {
			return
		}
	}
	resp = GetResponseInfo(qDoc, resID)
	//得等信息截图信息获取到了，才知道这个结果
	<-ch
	return
}
func GetHandleInfoKey(qid int64) (keyName string) {
	keyName = fmt.Sprintf("%s%d", HandleInfoBaseStr, qid)
	return
}

func SaveVideoInfoTemp(qid int64, doc *api.XngResourceInfoDoc) error {
	keyName := GetHandleInfoKey(qid)
	err := videoRedis.AddNewResInfoForHandle(keyName, doc)
	if err != nil {
		return err
	}
	return nil
}

func SubmitSnapShotFrame(sts *StsInfo, qid, snapQid int64, du float64) (err error) {
	//start := time.Now()
	var startTime int64
	if du > api.SnapConfTime {
		startTime = api.SnapConfTime
	} else {
		startTime = api.SnapConfShortTime
	}

	endPoint := sts.Data.EndpointInter
	if conf.Env != lib.PROD {
		endPoint = sts.Data.Endpoint
	}
	client, err := oss.New(endPoint, sts.Data.AccessKey, sts.Data.SecretKey, oss.SecurityToken(sts.Data.SecurityToken))
	if err != nil {
		return
	}
	objectName := strconv.FormatInt(qid, 10)
	srcBucket := sts.Data.InputBucket
	destBucket := conf.C.Bucket.Resource
	bucket, err := client.Bucket(srcBucket)
	if err != nil {
		return
	}
	targetImageName := strconv.FormatInt(snapQid, 10)
	style := fmt.Sprintf("video/snapshot,t_%d,f_jpg,m_fast,ar_auto", startTime)
	process := fmt.Sprintf("%s|sys/saveas,o_%v,b_%v", style, base64.URLEncoding.EncodeToString([]byte(targetImageName)), base64.URLEncoding.EncodeToString([]byte(destBucket)))
	resp, err := bucket.ProcessObject(objectName, process)
	if err != nil {
		//conf.Logger.Error("failed to snap cut frame, resp=%v, error=%v", resp, err)
		return errors.New(fmt.Sprintf("SubmitSnapShotFrame, err:%v, resp:%v", err, resp))
	}
	return
}

/*
func SubmitSnapShotSync(client *mts.Client, stsInfo *StsInfo, qid, snapQid int64, du float64) (err error) {
	//ch := make(chan int64, 1)
	//go GetDistributedId(ch)
	var startTime int64
	if du > api.SnapConfTime {
		startTime = api.SnapConfTime
	} else {
		startTime = api.SnapConfShortTime
	}

	destBucket := conf.C.Bucket.Resource
	srcBucket := stsInfo.Data.InputBucket
	location := stsInfo.Data.Endpoint[:len(stsInfo.Data.Endpoint)-13]
	snapName := strconv.FormatInt(snapQid, 10)
	videoName := strconv.FormatInt(qid, 10)
	request := mts.CreateSubmitSnapshotJobRequest()
	request.Scheme = "https"
	request.SnapshotConfig = fmt.Sprintf("{\"OutputFile\": {\"Bucket\": \"%s\",\"Location\":\"%s\",\"Object\": \"%s\"},\"Time\":\"%d\"}", destBucket, location, snapName, startTime)
	request.Input = fmt.Sprintf("{\"Bucket\":\"%s\", \"Location\": \"%s\",\"Object\":\"%s\" }", srcBucket, location, videoName)
	//request.PipelineId = api.SnapPipeLineID
	response, err := client.SubmitSnapshotJob(request)
	if err != nil {
		conf.Logger.Error("submit snap shot job error, request: %v, response: %v, error: %v", request, response, err)
		return
	}
	if response == nil {
		err = errors.New("error snap response is nil")
		return
	}
	return
}

func SubmitSnapShotAsync(client *mts.Client, stsInfo *StsInfo, qid, snapQid int64, du float64) (err error) {
	//ch := make(chan int64, 1)
	//go GetDistributedId(ch)
	var startTime int64
	if du > api.SnapConfTime {
		startTime = api.SnapConfTime
	} else {
		startTime = api.SnapConfShortTime
	}
	//snapQid = <-ch
	//if snapQid <= 0 {
	//	return errors.New("get new id for snap shot error")
	//}
	destBucket := conf.C.Bucket.Resource
	srcBucket := stsInfo.Data.InputBucket
	location := stsInfo.Data.Endpoint[:len(stsInfo.Data.Endpoint)-13]
	snapName := strconv.FormatInt(snapQid, 10)
	videoName := strconv.FormatInt(qid, 10)
	request := mts.CreateSubmitSnapshotJobRequest()
	request.Scheme = "https"
	request.SnapshotConfig = fmt.Sprintf("{\"OutputFile\": {\"Bucket\": \"%s\",\"Location\":\"%s\",\"Object\": \"%s\"},\"Time\":\"%d\",\"Num\":\"%d\",\"Interval\":\"%d\"}", destBucket, location, snapName, startTime, api.SnapConfNum, api.SnapConfInterval)
	request.Input = fmt.Sprintf("{\"Bucket\":\"%s\", \"Location\": \"%s\",\"Object\":\"%s\" }", srcBucket, location, videoName)
	request.PipelineId = api.SnapPipeLineID
	snapResp, err := client.SubmitSnapshotJob(request)
	if err != nil {
		conf.Logger.Error("submit snap shot job error, request: %v, response: %v, error: %v", request, snapResp, err)
		return
	}
	if snapResp == nil {
		return errors.New("error snap response is nil")
	}

	return
}

*/

func SetMetaContentType(fileName string, srcBucket string, stsInfo *StsInfo, resType string) error {
	endPoint := stsInfo.Data.EndpointInter
	if conf.Env != lib.PROD {
		endPoint = stsInfo.Data.Endpoint
	}
	client, err := oss.New(endPoint, stsInfo.Data.AccessKey, stsInfo.Data.SecretKey, oss.SecurityToken(stsInfo.Data.SecurityToken))
	if err != nil {
		return err
	}
	bucket, err := client.Bucket(srcBucket)
	if err != nil {
		return err
	}
	err = bucket.SetObjectMeta(fileName, oss.ContentType(resType))
	if err != nil {
		xlog.Error("file:%v set Content-Type:%v error, error = %v", fileName, resType, err)
		return err
	}
	return nil
}

func SetContentTypeForCommon(fileName string, stsInfo *sts.UploadToken, resType string) error {
	endPoint := stsInfo.EndpointInternal
	if conf.Env != lib.PROD {
		endPoint = stsInfo.Endpoint
	}
	client, err := oss.New(endPoint, stsInfo.AccessKey, stsInfo.SecretKey, oss.SecurityToken(stsInfo.SecurityToken))
	if err != nil {
		return err
	}
	bucket, err := client.Bucket(stsInfo.Bucket)
	if err != nil {
		return err
	}
	err = bucket.SetObjectMeta(fileName, oss.ContentType(resType))
	if err != nil {
		xlog.Error("file:%v set Content-Type:%v error, error = %v", fileName, resType, err)
		return err
	}
	return nil
}

//func GetNewResourceInfo(req api.MediaInfoReq, stsInfo *StsInfo) (api.XngResourceInfoDoc, error) {
//	regionID := stsInfo.Data.Endpoint[4 : len(stsInfo.Data.Endpoint)-13]
//	client, err := mts.NewClientWithStsToken(regionID, stsInfo.Data.AccessKey, stsInfo.Data.SecretKey, stsInfo.Data.SecurityToken)
//	if err != nil {
//		conf.Logger.Error("create new client with stsToken error", "error : ", err)
//		return api.XngResourceInfoDoc{}, err
//	}
//	uploadBucket := stsInfo.Data.InputBucket
//	mtsBucket := stsInfo.Data.OutputBucket
//	loc := stsInfo.Data.Endpoint[:len(stsInfo.Data.Endpoint)-13]
//	//conf.Logger.Info("Test", "time:", time.Now().Unix())
//	//获取视频资源信息
//	start := time.Now()
//	fileName := strconv.FormatInt(req.Qid, 10)
//	doc, err := GetVideoInfo(client, uploadBucket, loc, fileName)
//	if err != nil {
//		conf.Logger.Error("get video information error", "filename: ", req.Qid, "bucket: ", uploadBucket, "error: ", err)
//		return api.XngResourceInfoDoc{}, err
//	}
//	doc.ResId = req.Qid
//	doc.QeTag = req.QeTag
//	conf.Logger.Info("Test over get media info", "time:", time.Since(start))
//	start = time.Now()
//	//获取截图信息
//	snapQid, snapType, err := GetVideoScreenShot(client, doc.Du, uploadBucket, conf.C.Bucket.Resource, loc, fileName, req.Mid)
//	if err != nil {
//		conf.Logger.Error("get screen shot picture error", "filename: ", req.Qid, "bucket: ", mtsBucket, "mid: ", req.Mid, "error: ", err)
//		return api.XngResourceInfoDoc{}, err
//	}
//	doc.CoverTp = snapType
//	doc.Cover = snapQid
//	/*conf.Logger.Info("Test start snap meta", "time:", time.Since(start))
//	start = time.Now()
//	snapName := strconv.FormatInt(snapQid, 10)
//	err = SetMetaContentType(snapName, conf.C.Bucket.Resource, stsInfo, api.ContentTypeImg)
//	if err != nil {
//		conf.Logger.Error("set snap content-type error, qid=%v, error=%v", snapQid, err)
//		return api.XngResourceInfoDoc{}, err
//	}
//	conf.Logger.Info("Test over snap meta", "time:", time.Since(start))
//	*/
//	return doc, nil
//}

//func AddNewUserImgDoc(qid int64, mid int64) error {
//	ch := make(chan int64, 1)
//	go GetDistributedId(ch)
//	doc, err := GetResInfoByQid(qid)
//	if err != nil {
//		conf.Logger.Error("get resource info error", "qid: ", qid, "error: ", err)
//		return err
//	}
//	upt := time.Now().UnixNano() / 1e6
//	mt := upt
//	resId := <-ch
//	if resId <= 0 {
//		return errors.New("AddNewUserImgDoc fail to get new res id")
//	}
//	newDoc := &api.UserResourceDoc{
//		ResId: resId,
//		Size:  doc.Size,
//		QeTag: "",
//		Upt:   upt,
//		Mt:    mt,
//		Ct:    0,
//		Src:   doc.Src,
//		Fmt:   doc.Fmt,
//		Ort:   doc.Ort,
//		W:     doc.W,
//		H:     doc.H,
//		Qid:   qid,
//		Mid:   mid,
//		Ty:    doc.Type,
//	}
//	err = DaoByMid.AddResourceDoc(mid, newDoc)
//	if err != nil {
//		conf.Logger.Error("add resource info error", "qid: ", qid, "mid: ", mid, "error: ", err)
//		return err
//	}
//	return nil
//}

func UpdateResourceRecord(qDoc *api.XngResourceInfoDoc, qid int64, mid int64) (int64, error) {
	mDoc, err := DaoByMid.GetDocByMid(mid, qid)
	if err != nil {
		//conf.Logger.Error("get user resource doc by mid error", "mid: ", mid, "qid: ", qid, "error", err)
		return 0, err
	}
	var resID int64
	if mDoc == nil {
		//FIXME:并发
		doc, resId, err := CreateNewUserResDoc(qDoc, mid)
		if err != nil {
			xlog.Error("UpdateResourceRecord get new user resource doc error, mid:%v, qdoc:%v, err:%v", mid, qDoc, err)
			return 0, err
		}
		go func() {
			errIgnore := DaoByQid.UpdateResourceRef(qid)
			if errIgnore != nil {
				xlog.Error("UpdateResourceRecord DaoByQid UpdateResourceRef err:%v, qid:%v", errIgnore, qid)
			}
			listName := GetUserUpdateKeyName(qid)
			exists, errIgnore := videoRedis.ExistsResIdsList(listName)
			if errIgnore != nil {
				xlog.Error("UpdateResourceRecord ExistsResIdsList err:%v, key:%v", errIgnore, listName)
			}
			if !exists {
				return
			}
			if errIgnore = videoRedis.AddResIdToList(listName, resId, mid); errIgnore != nil {
				xlog.Error("UpdateResourceRecord AddResIdToList err, key:%v, resid:%v, mid:%v", listName, resId, mid)
			}
			return
		}()

		err = AddUserResDoc(doc, mid)
		if err != nil {
			xlog.Error("add user resource doc error:%v, qid:%v, mid:%v", err, qid, mid)
			return 0, err
		}
		//err = AddNewUserImgDoc(qDoc.Cover, mid)
		//if err != nil {
		//	conf.Logger.Error("failed to add user snap info, mid=%v, snapQid=%v, error=%v", mid, qDoc.Cover, err)
		//	return 0, err
		//}
		resID = resId
	} else {
		resID = mDoc.ResId
		err := UpdateUserResDoc(mDoc, mid)
		if err != nil {
			xlog.Error("update user resource doc error:%v, mid:%v", err, mid)
			return 0, err
		}
	}
	return resID, nil
}

func GetUserUpdateKeyName(qid int64) string {
	key := fmt.Sprintf("%s%d", UserUpdateBaseStr, qid)
	return key
}

func AddNewResource(doc *api.XngResourceInfoDoc, mid int64, qeTag string) (int64, error) {
	ch := make(chan int64, 1)
	go GetDistributedId(ch)
	//FIXME:这里面只有这两个字段？并发
	go func() {
		err := AddXngTagDoc(qeTag, doc.ResId)
		if err != nil {
			xlog.Error("add new tag doc error:%v, doc:%v, qetag:%v", err, doc, qeTag)
		}
		err = AddXngResDoc(doc, doc.ResId)
		if err != nil {
			xlog.Error("add new resource doc error:%v, doc:%v", err, doc)
		}
	}()

	resID := <-ch
	if resID <= 0 {
		return 0, errors.New("get new res id error")
	}
	nDoc := &api.UserResourceDoc{
		ResId: resID,
		Size:  doc.Size,
		QeTag: doc.QeTag,
		Upt:   doc.Upt,
		Mt:    doc.Mt,
		Ct:    doc.Ct,
		Src:   doc.Src,
		Fmt:   doc.Fmt,
		Ort:   doc.Ort,
		W:     doc.W,
		H:     doc.H,
		Qid:   doc.ResId,
		Mid:   mid,
		Du:    doc.Du,
		Cover: doc.Cover,
		Ty:    doc.Type,
		Code:  doc.Code,
	}
	err := AddUserResDoc(nDoc, mid)
	if err != nil {
		xlog.Error("add new user resource doc error:%v, doc:%v, mid:%v", err, doc, mid)
		return 0, err
	}
	keyName := GetUserUpdateKeyName(doc.ResId)
	err = videoRedis.AddResIdToList(keyName, resID, mid)
	if err != nil {
		xlog.Error("failed to add resId to temp list, key:%v, resid:%v, err:%v", keyName, resID, err)
		return 0, err
	}
	//err = videoRedis.AddUserResUpdateRecord(resID, mid)
	//if err != nil {
	//	conf.Logger.Error("failed to add user update record", "resId", resID, "mid", mid)
	//	return 0, err
	//}
	return nDoc.ResId, nil
}

//整理视频资源的响应内容
func GetResponseInfo(qDoc *api.XngResourceInfoDoc, resId int64) (resp *api.MediaInfoResp) {
	/*
		client, err := oss.New(stsInfo.Data.Endpoint, stsInfo.Data.AccessKey, stsInfo.Data.SecretKey, oss.SecurityToken(stsInfo.Data.SecurityToken))
		if err != nil {
			return api.MediaInfoResp{}, err
		}

		bucketForImg := stsInfo.Data.OutputBucket
			bucketForVideo := stsInfo.Data.InputBucket

			bucket, err := client.Bucket(bucketForVideo)
			if err != nil {
				return api.MediaInfoResp{}, err
			}

		imgBucket, err := client.Bucket(bucketForImg)
		if err != nil {
			return api.MediaInfoResp{}, err
		}
		//获取截图url
		picName := string(qdoc.Cover)
		picUrl, err := imgBucket.SignURL(picName, oss.HTTPGet, 60)
		if err != nil {
			return api.MediaInfoResp{}, err
		}

			//获取视频url
			videoUrl, err := bucket.SignURL(key, oss.HTTPGet, 600)
			if err != nil {
				return api.MediaInfoResp{}, err
			}*/
	resp = &api.MediaInfoResp{
		Id:    resId,
		Qid:   qDoc.ResId,
		Ty:    qDoc.Type,
		Size:  qDoc.Size,
		Upt:   qDoc.Upt,
		Mt:    qDoc.Mt,
		Ct:    qDoc.Ct,
		Src:   qDoc.Src,
		Fmt:   qDoc.Fmt,
		W:     qDoc.W,
		H:     qDoc.H,
		Du:    qDoc.Du,
		Cover: qDoc.Cover,
		Code:  qDoc.Code,
		QeTag: qDoc.QeTag,
	}
	return
}

/*
func ResNoNeedTrans(doc api.XngResourceInfoDoc, qid int64, stsInfo *StsInfo) (need bool, err error) {
	if doc.Size >= api.TransLimitSize {
		return true, nil
	}

	keyName := strconv.FormatInt(qid, 10)
	err = CopyResToBucket(keyName, stsInfo.Data.InputBucket, stsInfo.Data.OutputBucket, stsInfo)
	if err != nil {
		conf.Logger.Error("copy resource:%v below limit to bucket:%v error, err=%v", keyName, stsInfo.Data.OutputBucket, err)
		return
	}
	err = CopyResToBucket(keyName, stsInfo.Data.InputBucket, conf.C.Bucket.Resource, stsInfo)
	if err != nil {
		conf.Logger.Error("copy resource:%v below limit to bucket:%v error, err=%v", keyName, conf.C.Bucket.Resource, err)
		return
	}
	err = DelResFromBucket(keyName, stsInfo.Data.InputBucket, stsInfo)
	if err != nil {
		conf.Logger.Error("failed to delete resource on bucket:%v, error=%v", stsInfo.Data.InputBucket, err)
		return
	}
	qry := bson.M{"_id": qid}
	updata := bson.M{"$set": bson.M{"trans": ResStatusTrans}}

	err = DaoByQid.UpdateResourceDoc(qid, qry, updata)
	if err != nil {
		conf.Logger.Error("failed to update resource doc, qid=%v, err=%v", qid, err)
		return
	}
	return
}

*/

//提交转码（视频资源）
func (v *MediaVideo) TransCodeResource(req *api.TransCodeReq) (status int, err error) {
	//转码状态判断

	status, _, err = GetTransStatusWithInfo(req.Qid)
	if err != nil {
		xlog.Error("TransCodeResource get resource trans status error:%v, req:%v", err, req)
		return SubmitStatusFail, err
	}
	if status != TransStatusNot { //除了未转码状态的资源，其余都不做处理
		xlog.Info("resource is in status=%v, not submit trans code", status)
		return SubmitStatusSuccess, nil
	}

	stsInfo, err := GetStsForMts()
	if stsInfo == nil || err != nil {
		return SubmitStatusFail, errors.New(fmt.Sprintf("sts info is nil, err:%v", err))
	}

	//初始化请求信息
	//endPoint := stsInfo.Data.EndpointInter
	//if conf.Env != lib.PROD {
	//	endPoint = stsInfo.Data.Endpoint
	//}
	endPoint := stsInfo.Data.Endpoint

	regionID := endPoint[4 : len(endPoint)-13]
	client, err := mts.NewClientWithStsToken(regionID, stsInfo.Data.AccessKey, stsInfo.Data.SecretKey, stsInfo.Data.SecurityToken)
	if err != nil {
		xlog.Error("create new client with stsToken error:%v, accessKey:%v, secret:%v", err, stsInfo.Data.AccessKey, stsInfo.Data.SecretKey)
		return SubmitStatusFail, err
	}
	if conf.Env == lib.PROD {
		client.Domain = api.MtsVpcDomain
	}
	fileName := strconv.FormatInt(req.Qid, 10)
	userData, err := utils.SetMNSCallBackData(fileName, "", api.ResourceTypeVideo, 0, "", api.MTSJobTypeTransVideoForBigFile, fileName)
	if err != nil {
		xlog.Error("TransCodeResource.SetMNSCallBackData failed, id:[%s], err:[%v]", fileName, err)
		return SubmitStatusFail, err
	}
	request := mts.CreateSubmitJobsRequest()
	request.Scheme = "http"
	request.Outputs = fmt.Sprintf("[{\"OutputObject\":\"%s\",\"TemplateId\":\"%s\",\"UserData\":\"%s\"}]", fileName, api.TransCodeTemplateId, userData)
	request.OutputBucket = stsInfo.Data.OutputBucket
	request.PipelineId = utils.GetMTSPipeId(conf.Env)
	//if conf.Env != lib.PROD {
	//	request.PipelineId = api.TransCodePipeIdForTest
	//}
	location := endPoint[:len(endPoint)-13]
	request.Input = fmt.Sprintf("{\"Bucket\":\"%s\",\"Location\":\"%s\",\"Object\":\"%s\"}", stsInfo.Data.InputBucket, location, fileName)
	request.OutputLocation = location
	xlog.Info("submit trans job, request: %v", request)
	//提交转码作业，并添加临时记录到redis内
	response, err := client.SubmitJobs(request)
	if err != nil || response == nil {
		xlog.Error("submit transCode job error, qid:%v, response:%v", req.Qid, response)
		return SubmitStatusFail, err
	}
	if response.JobResultList.JobResult[0].Success == false {
		xlog.Info("submit failed")
		return SubmitStatusFail, errors.New("submit video transCode failed")
	}
	jobId := response.JobResultList.JobResult[0].Job.JobId
	//用于同一时间段重复提交转码的校验
	//for i := 0; i < api.ReTryTimes; i++ {
	err = videoRedis.AddResourceTransRecord(req.Qid, jobId)
	if err != nil {
		xlog.Error("add resource trans temp record error:%v, qid:%v", err, req.Qid)
		return SubmitStatusFail, err
	}
	xlog.Info("add resource trans record qid:%v, jobid:%v", req.Qid, response.JobResultList.JobResult[0].Job.JobId)

	return SubmitStatusSuccess, nil
}

//生成指定上传对象
func ResourceFactory(resType int) ResourceModel {
	switch resType {
	case api.ResourceTypeVideo:
		return &MediaVideo{resType: api.ResourceTypeVideo}
	default:
		//conf.Logger.Error("Wrong type of uploaded resource, type: %v", resType) //FIXME:参数打出来 xlog.Error("Wrong type of uploaded resource, type:%v", resType)
		return nil
	}
}

// ExistResByQeTag 利用qeTag验证对于qid是否存在，上传完才会插入
func ExistResByQeTag(qeTag string) (bool, error) {
	doc, err := DaoByTag.GetDocByTag(qeTag)
	if err != nil {
		return false, err
	}
	if doc == nil {
		return false, nil
	}
	qDoc, err := DaoByQid.GetDocByQid(doc.Qid)
	if err != nil {
		return false, err
	}
	if qDoc == nil {
		return false, nil
	}
	return true, nil
}

//资源上传状态的校验，即重复校验
//返回值资源为上传的状态（0为未上传，1为上传中，2为已上传
func GetUploadStatus(req *api.UploadStatusReq) (status int, err error) {
	//上传中判断
	//var isUpload bool

	//isUpload, err = videoRedis.IsUploading(req.QeTag)
	//if err != nil {
	//	//conf.Logger.Error("GetUploadStatus: is uploading err", "req", *req, "err", err)
	//	return
	//}
	//
	//if isUpload == true {
	//	status = UploadStatusTrying
	//	return
	//}
	//资源是否已经存在，已上传
	isExists, err := ExistResByQeTag(req.QeTag)
	if err != nil {
		//conf.Logger.Error("GetUploadStatus: exists res by qeTag err", "req", *req, "err", err)
		return
	}
	if isExists == true {
		status = UploadStatusSuccess
		return
	}
	status = UploadStatusNot
	return
}

//设置上传状态
func SetUploadStatus(req *api.SetUploadStatusReq, ty int) (err error) {

	switch req.Status {
	case UploadStatusTrying:
		//err = videoRedis.AddUploadingRecord(req.QeTag, req.Qid)
		//if err != nil {
		//	return
		//}
	case UploadStatusFail, UploadStatusCancel:
		//err = videoRedis.DelUploadingRecord(req.QeTag)
		//if err != nil {
		//	return
		//}
	case UploadStatusSuccess:
		//err = videoRedis.DelUploadingRecord(req.QeTag)
		//if err != nil {
		//	return
		//}
		model := ResourceFactory(ty)
		if model == nil {
			err = errors.New("create ResourceFactory err")
			return
		}
		err = model.HandleNewResource(req)
		if err != nil {
			return
		}

	default:
		xlog.Error("set upload status error, unknown status, req:%v", req)
	}
	return
}

//获取临时凭证信息
func GetStsVoucher() (ret StsInfoForUser, err error) {
	stsUrl := api.XngStsVoucherUrlForProd
	if conf.Env != lib.PROD {
		stsUrl = api.XngStsVoucherUrlForTest
	}
	req, err := http.NewRequest(`POST`, stsUrl, nil)
	if err != nil {
		return
	}
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	result, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(result, &ret)
	if err != nil {
		return ret, err
	}
	return
}

func GetStsForMts() (ret *StsInfo, err error) {
	mutex.Lock()
	if GlobalStsInfoForMts != nil {
		ret = GlobalStsInfoForMts
		mutex.Unlock()
		return
	}
	mutex.Unlock()
	ret, err = getStsForMtsFromRemote()
	if ret != nil && err == nil {
		mutex.Lock()
		GlobalStsInfoForMts = ret
		mutex.Unlock()
	}
	return
}

func getStsForMtsFromRemote() (ret *StsInfo, err error) {
	stsUrl := api.XngStsForMtsUrlForProd
	if conf.Env != lib.PROD {
		stsUrl = api.XngStsForMtsUrlForTest
	}
	req, err := http.NewRequest(`POST`, stsUrl, nil)
	if err != nil {
		return
	}
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	if resp == nil {
		err = errors.New("get sts info for mts option error, resp is nil")
		return
	}
	result, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(result, &ret)
	if err != nil {
		return
	}
	return
}

//FIXME:这个获取自增id的这个不自己写了，有个获取自增id的服务，使用http请求那个服务，注意源代码中用的是哪个自增id库表
//生成自增ID，通过http请求自增id的服务，获取新的id
//TODO:定时获取存到channel里面
func GetDistributedId(ch chan int64) {

	var id int64
	var ok bool
	for i := 0; i < api.ReTryTimes; i++ {
		resp, err := ids_api.GetNewIds(conf.C.Addrs.Ids, "xng-res") //表名为res，主键为res
		if err != nil {
			xlog.Error("get new distributed id error:%v", err)
			//ch <- 0
			continue
		}
		if resp == nil {
			xlog.Error("get new distributed id error, resp nil")
			//ch <- 0
			continue
		}
		id, ok = resp.Data["id"]
		if ok == false {
			xlog.Error("get new distributed id error, resp data nil resp:%v", resp.Data)
			//ch <- 0
			id = 0
			continue
		}
		break
	}
	ch <- id
}

// 获取临时上传凭证
func TempVoucher() (resp *api.TempVoucherResp, err error) {
	ch := make(chan int64, 1)
	go GetDistributedId(ch)
	stsInfo, err := GetStsVoucher()
	if err != nil {
		return
	}
	qid := <-ch
	if qid <= 0 {
		err = errors.New("TempVoucher get new distributed id error")
		return
	}
	ind := strings.Index(stsInfo.Data.Endpoint, ".")
	region := stsInfo.Data.Endpoint[0:ind]
	resp = &api.TempVoucherResp{
		Endpoint:      stsInfo.Data.Endpoint,
		EndpointInter: stsInfo.Data.EndpointInter,
		Bucket:        stsInfo.Data.Bucket,
		AccessKey:     stsInfo.Data.AccessKey,
		SecretKey:     stsInfo.Data.SecretKey,
		SecurityToken: stsInfo.Data.SecurityToken,
		Region:        region,
		Qid:           qid,
	}
	return
}

//添加用户资源记录
func AddUserResDoc(doc *api.UserResourceDoc, mid int64) error {
	var err error
	for i := 0; i < api.ReTryTimes; i++ {
		err = videoRedis.IncreaseUserAccount(mid)
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}
	err = DaoByMid.AddResourceDoc(mid, doc)
	if err != nil {
		return err
	}
	return nil
}

//添加重复校验的etag记录
func AddXngTagDoc(qetag string, qid int64) error {
	err := DaoByTag.AddXngTagDoc(qetag, qid)
	if err != nil {
		return err
	}
	return nil
}

//添加xng资源记录
func AddXngResDoc(doc *api.XngResourceInfoDoc, qid int64) error {
	err := DaoByQid.InsertResourceDoc(qid, doc)
	if err != nil {
		return err
	}
	return nil
}

/*
//初始化截图请求内容
func InitSubmitSnapRequest(srcBucket string, destBucket string, location string, snapId int64, videoName string, du float64) (*mts.SubmitSnapshotJobRequest, error) {
	if srcBucket == "" || destBucket == "" || location == "" {
		return nil, errors.New("init SnapShot params error")
	}
	snapName := strconv.FormatInt(snapId, 10)
	request := mts.CreateSubmitSnapshotJobRequest()
	request.Scheme = "https"
	var ntime int
	if du > api.VideoDurationForSnapShot {
		ntime = api.SnapConfTime
	} else {
		ntime = api.SnapConfShortTime
	}
	request.SnapshotConfig = fmt.Sprintf("{\"OutputFile\": {\"Bucket\": \"%s\",\"Location\":\"%s\",\"Object\": \"%s\"},\"Time\":\"%d\",\"Num\":\"%d\",\"Interval\":\"%d\"}", destBucket, location, snapName, ntime, api.SnapConfNum, api.SnapConfInterval)
	request.Input = fmt.Sprintf("{\"Bucket\":\"%s\", \"Location\": \"%s\",\"Object\":\"%s\" }", srcBucket, location, videoName)
	request.PipelineId = api.SnapPipeLineID

	return request, nil
}



//初始化查询截图作业请求
func InitQuerySnapRequet(jobId string) (*mts.QuerySnapshotJobListRequest, error) {
	if jobId == "" {
		return nil, errors.New("init QuerySnapShot params error, jobId=nil")
	}
	request := mts.CreateQuerySnapshotJobListRequest()
	request.Scheme = "https"
	request.SnapshotJobIds = jobId
	return request, nil
}

*/

//获取视频的截图信息，返回截图的qid，截图资源类型
//func GetVideoScreenShot(client *mts.Client, du float64, srcBucket string, destBucket string, location string, key string, mid int64) (int64, string, error) {
//	//获取自增ID作为截图的资源ID
//	ch := make(chan int64, 1)
//	go GetDistributedId(ch)
//	snapQid := <-ch
//	if snapQid <= 0 {
//		return 0, "", errors.New("get new id for snap shot error")
//	}
//	go GetDistributedId(ch)
//	snapReq, err := InitSubmitSnapRequest(srcBucket, destBucket, location, snapQid, key, du)
//	if err != nil {
//		return 0, "", err
//	}
//	now := time.Now()
//	//提交截图作业
//	snapResp, err := client.SubmitSnapshotJob(snapReq)
//	if err != nil {
//		conf.Logger.Error("submit snap shot job error, request: %v, response: %v, error: %v", snapReq, snapResp, err)
//		return 0, "", err
//	}
//	if snapResp == nil {
//		return 0, "", errors.New("error snap response is nil")
//	}
//	snapJobId := snapResp.SnapshotJob.Id
//
//	querySnapReq, err := InitQuerySnapRequet(snapJobId)
//	if err != nil {
//		conf.Logger.Error("init query snap request error", "jobID: ", snapJobId, "error: ", err)
//		return 0, "", err
//	}
//	i := 0
//	for i < QuerySnapTryTime {
//		//time.Sleep(time.Duration(1) * time.Second)
//		querySnapResp, err := client.QuerySnapshotJobList(querySnapReq)
//		if err != nil {
//			conf.Logger.Error("query snap shot job error, request: %v, response: %v, error: %v", querySnapReq, querySnapResp, err)
//			return 0, "", err
//		}
//		if querySnapResp.SnapshotJobList.SnapshotJob[0].State == "Success" {
//			break
//		}
//		if querySnapResp.SnapshotJobList.SnapshotJob[0].State == "Fail" {
//			return 0, "", errors.New("submit snapshot job error")
//		}
//		time.Sleep(time.Millisecond * 10)
//		i++
//	}
//	if i >= QuerySnapTryTime {
//		return 0, "", errors.New("submit snapshot job out of time")
//	}
//	conf.Logger.Info("Test over snap shot", "time:", time.Since(now))
//	//截图成功后获取截图信息
//	now = time.Now()
//	request := mts.CreateSubmitMediaInfoJobRequest()
//	request.Scheme = "https"
//	request.Async = requests.NewBoolean(false)
//	picName := strconv.FormatInt(snapQid, 10)
//	request.Input = fmt.Sprintf("{\"Bucket\":\"%s\",\"Location\":\"%s\",\"Object\":\"%s\"}", destBucket, location, picName)
//	response, err := client.SubmitMediaInfoJob(request)
//	if err != nil {
//		conf.Logger.Error("submit media info job error, request: %v, response: %v, error: %v", request, response, err)
//		return 0, "", err
//	}
//	if response == nil {
//		return 0, "", errors.New("mediaInfo job response is nil")
//	}
//	conf.Logger.Info("Test over snap info", "time:", time.Since(now))
//	fsize, err := strconv.ParseInt(response.MediaInfoJob.Properties.FileSize, 10, 64)
//	if err != nil {
//		conf.Logger.Error("failed to change string to int", "resp", response.MediaInfoJob.Properties)
//		return 0, "", err
//	}
//	width, err := strconv.Atoi(response.MediaInfoJob.Properties.Width)
//	if err != nil {
//		return 0, "", err
//	}
//	height, err := strconv.Atoi(response.MediaInfoJob.Properties.Height)
//	if err != nil {
//		return 0, "", err
//	}
//	upt := time.Now().UnixNano() / 1e6
//	mt := upt
//	qdoc := &api.XngResourceInfoDoc{
//		ResId: snapQid,
//		Type:  api.ResourceTypeImg,
//		Size:  fsize,
//		Upt:   upt,
//		Mt:    mt,
//		Ct:    0,
//		Src:   api.UploadFromWXMiniApp,
//		Fmt:   api.SnapShotType,
//		Ort:   1,
//		W:     width,
//		H:     height,
//		Ref:   1,
//	}
//
//	//截图信息添加入库
//	err = DaoByQid.InsertResourceDoc(snapQid, qdoc)
//	if err != nil {
//		conf.Logger.Error("add new resource doc error, qid: %v, doc: %v, error: %v", snapQid, qdoc, err)
//		return 0, "", err
//	}
//	//截图信息添加到用户资源库
//	newId := <-ch
//	if newId <= 0 {
//		return 0, "", errors.New("get new id for snap user resource error")
//	}
//	mdoc := &api.UserResourceDoc{
//		ResId: newId,
//		Size:  fsize,
//		Upt:   upt,
//		Mt:    mt,
//		Ct:    0,
//		Src:   api.UploadFromWXMiniApp,
//		Fmt:   api.SnapShotType,
//		Ort:   1,
//		W:     width,
//		H:     height,
//		Qid:   snapQid,
//		Mid:   mid,
//		Ty:    api.ResourceTypeImg,
//		Dt:    0,
//		D:     0,
//	}
//	err = DaoByMid.AddResourceDoc(mid, mdoc)
//	if err != nil {
//		conf.Logger.Error("add user resource doc error, mid: %v, doc: %v, error: %v", mid, mdoc, err)
//		return 0, "", err
//	}
//	return snapQid, "img", nil
//}

//提交作业获取视频资源相关信息
func GetVideoInfo(client *mts.Client, Bucket string, location string, key string) (doc *api.XngResourceInfoDoc, err error) {
	request := mts.CreateSubmitMediaInfoJobRequest()
	request.Scheme = "https"
	request.Async = requests.NewBoolean(false)

	request.Input = fmt.Sprintf("{\"Bucket\":\"%s\",\"Location\":\"%s\",\"Object\":\"%s\"}", Bucket, location, key)
	//提交媒体信息作业
	response, err := client.SubmitMediaInfoJob(request)
	if err != nil {
		//conf.Logger.Error("submit media info job error, request: %v, response: %v, error: %v", request, response, err)
		return
	}
	if response == nil {
		err = errors.New("submit media info job error, response nil")
		return
	}
	if response.MediaInfoJob.State == "Fail" {
		xlog.Error("submit media info job error, resp:%v, key:%v", response, key)
		err = errors.New("submit media info job failed")
		return
	}
	upt := time.Now().UnixNano() / 1e6
	mt := upt
	qid, err := strconv.ParseInt(key, 10, 64)
	if err != nil {
		return
	}
	fSize, err := strconv.ParseInt(response.MediaInfoJob.Properties.FileSize, 10, 64)
	if err != nil {
		return
	}
	width, err := strconv.Atoi(response.MediaInfoJob.Properties.Width)
	if err != nil {
		return
	}
	height, err := strconv.Atoi(response.MediaInfoJob.Properties.Height)
	if err != nil {
		return
	}
	duration, err := strconv.ParseFloat(response.MediaInfoJob.Properties.Duration, 64)
	if err != nil {
		return
	}
	duration = duration * 1000

	doc = &api.XngResourceInfoDoc{
		ResId:     qid,
		Type:      api.ResourceTypeVideo,
		Size:      fSize,
		Upt:       upt,
		Mt:        mt,
		Ct:        0,
		Src:       api.UploadFromWXMiniApp,
		Fmt:       response.MediaInfoJob.Properties.Format.FormatName,
		W:         width,
		H:         height,
		Ref:       1,
		Du:        duration,
		TransCode: &ResStatusNot,
		Code:      response.MediaInfoJob.Properties.Streams.VideoStreamList.VideoStream[0].CodecName,
	}
	return
}

//修改用户资源的记录
func UpdateUserResDoc(doc *api.UserResourceDoc, mid int64) error {
	qry := bson.M{"_id": doc.ResId, "mid": mid}
	var upData bson.M
	upt := time.Now().UnixNano() / 1e6
	if doc.D == 0 {
		upData = bson.M{"$set": bson.M{"upt": upt}} //update注意
	} else {
		upData = bson.M{"$unset": bson.M{"d": 1}, "$set": bson.M{"upt": upt}}
	}
	err := DaoByMid.UpdateUserResourceDoc(qry, upData, mid)
	if err != nil {
		//conf.Logger.Error("update user resource error, mid: %v, updata: %v, error: %v", mid, upData, err)
		return err
	}
	return nil
}

//整理出资源信息的文档
func CreateNewUserResDoc(qdoc *api.XngResourceInfoDoc, mid int64) (reDoc *api.UserResourceDoc, resId int64, err error) {
	ch := make(chan int64, 1)
	go GetDistributedId(ch)
	resId = <-ch
	if resId <= 0 {
		err = errors.New("get new distributed id error")
		return
	}
	ct := time.Now().UnixNano() / 1e6
	upt := ct
	mt := upt
	reDoc = &api.UserResourceDoc{
		ResId: resId,
		Size:  qdoc.Size,
		QeTag: qdoc.QeTag,
		Upt:   upt,
		Mt:    mt,
		Ct:    ct,
		Src:   qdoc.Src,
		Fmt:   qdoc.Fmt,
		Ort:   qdoc.Ort,
		W:     qdoc.W,
		H:     qdoc.H,
		Qid:   qdoc.ResId,
		Mid:   mid,
		Du:    qdoc.Du,
		Cover: qdoc.Cover,
		Ty:    qdoc.Type,
		Code:  qdoc.Code,
	}
	return
}

//获取不同类型媒体资源的信息
func MediaInfo(req *api.MediaInfoReq, ty int) (resp *api.MediaInfoResp, err error) {
	//获取qid对应的资源信息
	model := ResourceFactory(ty)
	if model == nil {
		err = errors.New("create uploadModel error")
		return
	}
	resp, err = model.GetResourceInfo(req)
	if err != nil {
		return
	}
	return
}

//判断是否曾经转码
func GetResInfoByQid(qid int64) (doc *api.XngResourceInfoDoc, err error) {
	doc, err = DaoByQid.GetDocByQid(qid)
	if err != nil {
		//conf.Logger.Error("get doc by qid error, qid: ", qid, "error: ", err)
		return
	}
	return
}

func CopyVideoResToBucket(filename string, srcBucket string, destBucket string, stsInfo *StsInfo, totalSize int64) error {
	if totalSize <= 0 {
		xlog.Error("copy resource error, file size = %v", totalSize)
		return errors.New("error file size <= 0")
	}
	endPoint := stsInfo.Data.EndpointInter
	if conf.Env != lib.PROD {
		endPoint = stsInfo.Data.Endpoint
	}
	client, err := oss.New(endPoint, stsInfo.Data.AccessKey, stsInfo.Data.SecretKey, oss.SecurityToken(stsInfo.Data.SecurityToken))
	if err != nil {
		return err
	}
	//分片拷贝
	bucket, err := client.Bucket(destBucket)
	if err != nil {
		return err
	}
	imur, err := bucket.InitiateMultipartUpload(filename)
	if err != nil {
		xlog.Error("init multiPartUpload error, fileName=%v, error=%v", filename, err)
		return err
	}
	var partSize int64
	partSize = api.CopyResCutSize
	var chunkN = totalSize / partSize
	var chunks []oss.FileChunk
	var chunk = oss.FileChunk{}
	for i := int64(0); i < chunkN; i++ {
		chunk.Number = int(i + 1)
		chunk.Offset = i * partSize
		chunk.Size = partSize
		chunks = append(chunks, chunk)
	}
	if totalSize%partSize > 0 {
		chunk.Number = len(chunks) + 1
		chunk.Offset = int64(len(chunks)) * partSize
		chunk.Size = totalSize % partSize
		chunks = append(chunks, chunk)
	}

	var parts []oss.UploadPart
	for _, chunckd := range chunks {
		part, err := bucket.UploadPartCopy(imur, srcBucket, filename, chunckd.Offset, chunckd.Size, chunckd.Number)
		if err != nil {
			xlog.Error("copy resource:%v to bucket:%v, error=%v", filename, destBucket, err)
			return err
		}
		parts = append(parts, part)
	}
	_, err = bucket.CompleteMultipartUpload(imur, parts)
	if err != nil {
		xlog.Error("copy resource:%v to bucket:%v, error=%v", filename, destBucket, err)
		return err
	}
	return nil
}

func CopyLittleResToBucket(qidName string, srcBucket string, destBucket string, stsInfo *StsInfo) error {
	endPoint := stsInfo.Data.EndpointInter
	if conf.Env != lib.PROD {
		endPoint = stsInfo.Data.Endpoint
	}
	client, err := oss.New(endPoint, stsInfo.Data.AccessKey, stsInfo.Data.SecretKey, oss.SecurityToken(stsInfo.Data.SecurityToken))
	if err != nil {
		return err
	}
	bucket, err := client.Bucket(srcBucket)
	if err != nil {
		return err
	}
	_, err = bucket.CopyObjectTo(destBucket, qidName, qidName)
	if err != nil {
		xlog.Error("copy resource from %v bucket to %v bucket, error = %v", destBucket, srcBucket, err)
		return err
	}
	return nil
}

func DelResFromBucket(filename string, srcBucket string, stsInfo *StsInfo) error {
	endPoint := stsInfo.Data.EndpointInter
	if conf.Env != lib.PROD {
		endPoint = stsInfo.Data.Endpoint
	}
	client, err := oss.New(endPoint, stsInfo.Data.AccessKey, stsInfo.Data.SecretKey, oss.SecurityToken(stsInfo.Data.SecurityToken))
	if err != nil {
		return err
	}
	bucket, err := client.Bucket(srcBucket)
	if err != nil {
		return err
	}
	err = bucket.DeleteObject(filename)
	if err != nil {
		return err
	}
	return nil
}

func UpdateTransInfo(qid int64, doc *api.XngResourceInfoDoc, status int) (err error) {
	//keyName := strconv.FormatInt(qid, 10)
	////endPoint := stsInfo.Data.EndpointInter
	////if conf.Env != lib.PROD {
	////	endPoint = stsInfo.Data.Endpoint
	////}
	//endPoint := stsInfo.Data.Endpoint
	//regionID := endPoint[4 : len(endPoint)-13]
	//client, err := mts.NewClientWithStsToken(regionID, stsInfo.Data.AccessKey, stsInfo.Data.SecretKey, stsInfo.Data.SecurityToken)
	//if err != nil {
	//	conf.Logger.Error("get mts new client error", "accessKey: ", stsInfo.Data.AccessKey, "secretKey: ", stsInfo.Data.SecretKey, "token: ", stsInfo.Data.SecurityToken, "error: ", err)
	//	return
	//}
	//if conf.Env == lib.PROD {
	//	client.Domain = api.MtsVpcDomain
	//}
	//loc := endPoint[:len(endPoint)-13]
	//doc, err = GetVideoInfo(client, stsInfo.Data.OutputBucket, loc, keyName)
	//if err != nil || doc == nil {
	//	conf.Logger.Error("get video information error", "filename: ", keyName, "error: ", err)
	//	return
	//}

	//更新资源库内信息 todo:这块也提出去，函数太杂糅了，提不出去就再写个几个函数封装
	qry := bson.M{"_id": qid}
	mt := time.Now().UnixNano() / 1e6
	updata := bson.M{"$set": bson.M{"trans": status, "mt": mt, "size": doc.Size, "w": doc.W, "h": doc.H, "code": doc.Code}}
	err = DaoByQid.UpdateResourceDoc(qid, qry, updata)
	if err != nil {
		xlog.Error("update resource doc error:%v, qid:%v", qid, err)
		return
	}
	//conf.Logger.Info("update xng resource doc success, data=%v", updata)

	listName := GetUserUpdateKeyName(qid)
	resIdMids, err := videoRedis.GetResIdsFromList(listName)
	if err != nil {
		xlog.Error("failed to get resIds from list, qid=%v, error=%v", listName, err)
		return
	}
	upData := bson.M{"$set": bson.M{"size": doc.Size, "fmt": doc.Fmt, "mt": mt, "w": doc.W, "h": doc.H, "code": doc.Code}}
	for _, resIdMid := range resIdMids {
		resIdMidArr := strings.Split(resIdMid, ":")
		if len(resIdMidArr) != 2 {
			xlog.Error("fail to split resIdMid resIdMid:%v", resIdMid)
			continue
		}
		id, errIgnore := strconv.ParseInt(resIdMidArr[0], 10, 64)
		mid, errIgnore := strconv.ParseInt(resIdMidArr[1], 10, 64)
		if errIgnore != nil {
			xlog.Error("fail to parseInt resIdMid resIdMid:%v", resIdMid)
			continue
		}
		qry := bson.M{"_id": id, "qid": qid}
		//mid, nerr := videoRedis.GetUserResUpdateRecord(id)
		//if nerr != nil {
		//	conf.Logger.Error("failed to get user resource update record, resId=%v, mid=%v, error=%v", id, mid, nerr)
		//	continue
		//}
		errIgnore = DaoByMid.UpdateUserResourceDoc(qry, upData, mid)
		if errIgnore != nil {
			xlog.Error("failed to update user resource doc, qryData=%v, resId=%v, mid=%v, error=%v", qry, id, mid, errIgnore)
			continue
		}
		//nerr = videoRedis.DelUserResUpdateRecord(id)
		//if nerr != nil {
		//	conf.Logger.Error("failed to delete user resource update record, resId=%v, mid=%v, error=%v", id, mid, nerr)
		//	continue
		//}
	}
	err = videoRedis.DelUserResIdsList(listName)
	if err != nil {
		xlog.Error("delete user resource update list error, doc=%v, error=%v", doc, err)
		return
	}
	xlog.Info("update user resource doc success, data=%v", upData)
	return
}

//阿里云回调处理函数
func DealCallBack(data *api.MNSMessageData) error {
	if data.Type != api.NotifyTypeTrans {
		return nil
	}
	param, err := utils.GetMNSCallBackData(context.Background(), data.UserData)
	if err != nil {
		return err
	}
	if param.JobType != api.MTSJobTypeTransVideoForBigFile {
		return nil
	}
	qid, err := strconv.ParseInt(param.UserData, 10, 64)
	if err != nil {
		return err
	}
	//获取用户等待列表
	//var waitCnt int
	listKey := GetTransListKey(qid)
	//waitCnt, err = videoRedis.GetTransListCnt(listKey)
	//if err != nil {
	//	conf.Logger.Error("get wait user record error, jobID=%v, error=%v", data.JobID, err)
	//	return err
	//}

	keyName := strconv.FormatInt(qid, 10)
	//转码成功
	var doc *api.XngResourceInfoDoc
	ch := make(chan interface{}, 1)
	chRes := make(chan interface{}, 1)
	stsInfo, err := GetStsForMts()
	if stsInfo == nil || err != nil {
		return errors.New(fmt.Sprintf("sts info is nil, err:%v", err))
	}
	if data.State == api.NotifyStatusSuccess {

		transType := ResStatusTrans
		//keyName := strconv.FormatInt(qid, 10)
		endPoint := stsInfo.Data.Endpoint
		regionID := endPoint[4 : len(endPoint)-13]
		client, err := mts.NewClientWithStsToken(regionID, stsInfo.Data.AccessKey, stsInfo.Data.SecretKey, stsInfo.Data.SecurityToken)
		if err != nil {
			xlog.Error("get mts new client error:%v, accessKey: %v, secretKey:%v, token: %v", err, stsInfo.Data.AccessKey, stsInfo.Data.SecretKey, stsInfo.Data.SecurityToken)
			return err
		}
		if conf.Env == lib.PROD {
			client.Domain = api.MtsVpcDomain
		}
		loc := endPoint[:len(endPoint)-13]
		doc, err = GetVideoInfo(client, stsInfo.Data.OutputBucket, loc, keyName)
		if err != nil || doc == nil {
			xlog.Error("get video information error:%v, filename: %v", err, keyName)
			err = errors.New(fmt.Sprintf("callbakc get video info err:%v or doc is nil", err))
			return err
		}
		go func() {
			errIgnore := CopyVideoResToBucket(keyName, stsInfo.Data.OutputBucket, conf.C.Bucket.Resource, stsInfo, doc.Size)
			if errIgnore != nil {
				xlog.Error("copy resource to resource bucket error:%v, fileName: %v", errIgnore, keyName)
				chRes <- errIgnore
				return
			}
			chRes <- struct{}{}
			errIgnore = SetMetaContentType(keyName, conf.C.Bucket.Resource, stsInfo, api.ContentTypeVideo)
			if errIgnore != nil {
				//chRes <- errIgnore
				xlog.Error("set resource content-type error:%v, fileName:%v, type:%v, bucket:%v", errIgnore, keyName, api.ContentTypeVideo, errIgnore, conf.C.Bucket.Resource)
				//return
			}
		}()
		err = UpdateTransInfo(qid, doc, transType)
		if err != nil {
			xlog.Error("update trans resource information error:%v, qid:%v", err, qid)
			return err
		}

		//删除未转码的资源
		//err = DelResFromBucket(keyName, stsInfo.Data.InputBucket, stsInfo)
		//if err != nil {
		//	conf.Logger.Error("delete resource from upload bucket error", "fileName: ", keyName, "error: ", err)
		//	return err
		//}
	} else {
		chRes <- struct{}{}
	}

	//删除redis临时记录
	for i := 0; i < api.ReTryTimes; i++ {
		err = videoRedis.DelResourceTransRecord(qid)
		if err == nil {
			break
		}
	}
	if err != nil {
		xlog.Error("delete resource trans record error:%v, qid:%v", err, qid)
		return err
	}
	//if waitCnt <= 0 {
	//	conf.Logger.Info("waitCnt <= 0", "data", data)
	//	return nil
	//}

	var aids []int64
	for i := 0; i < api.ReTryTimes; i++ {
		aids, err = videoRedis.GetAidsFromTransList(listKey)
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}

	resResultCh := <-chRes
	switch result := resResultCh.(type) {
	case struct{}:
		xlog.Debug("succ copy video to res bucket")
	case error:
		return result
	default:
		return errors.New(fmt.Sprintf("unkown res notice type:%v", resResultCh))
	}

	if len(aids) <= 0 {
		xlog.Info("aids is nil, data:%v", data)
		return nil
	}

	//拷贝一份资源到影集bucket
	if data.State == api.NotifyStatusSuccess && doc != nil {
		go func() {
			errIgnore := CopyVideoResToBucket(keyName, stsInfo.Data.OutputBucket, conf.C.Bucket.Album, stsInfo, doc.Size)
			if errIgnore != nil {
				xlog.Error("copy resource to album bucket error:%v, fileName:%v", errIgnore, keyName)
				ch <- errIgnore
				return
			}
			qry := bson.M{"_id": qid}
			mt := time.Now().UnixNano() / 1e6
			upData := bson.M{"$set": bson.M{"trans": ResStatusMade, "mt": mt}}
			errIgnore = DaoByQid.UpdateResourceDoc(qid, qry, upData)
			if errIgnore != nil {
				xlog.Error("update resource doc error:%v, qid:%v", errIgnore, qid)
				ch <- errIgnore
				return
			}
			errIgnore = SetMetaContentType(keyName, conf.C.Bucket.Album, stsInfo, api.ContentTypeVideo)
			if errIgnore != nil {
				xlog.Error("set resource content-type error:%v", errIgnore)
				//return err
				ch <- errIgnore
				return
			}
			ch <- struct{}{}
			//删除临时bucket内的资源
			errIgnore = DelResFromBucket(keyName, stsInfo.Data.OutputBucket, stsInfo)
			if errIgnore != nil {
				xlog.Error("delete resource from upload bucket error:%v, fileName:%v", errIgnore, keyName)
				//return err
			}
		}()
	}

	for i := 0; i < api.ReTryTimes; i++ {
		err = videoRedis.DelAidsFromTransList(listKey)
		if err == nil {
			break
		}
	}
	if err != nil {
		xlog.Error("delete aids from list error:%v, qid:%v", err, qid)
		return err
	}
	if data.State == api.NotifyStatusSuccess && doc != nil {
		//等待影集bucket拷贝完成
		noticeCh := <-ch
		switch result := noticeCh.(type) {
		case struct{}:
			xlog.Debug("succ copy video to album bucket")
		case error:
			return result
		default:
			return errors.New(fmt.Sprintf("unkown notice type:%v", noticeCh))
		}
		for _, id := range aids {
			err = PushSuccessMessage(doc, id)
			if err != nil {
				xlog.Error("push success message to client error:%v, qid:%v ,aid:%v", err, qid, id)
				continue
			}
		}
		return nil
	}

	for _, id := range aids {
		err = PushFailMessage(id)
		if err != nil {
			xlog.Error("push fail message to client error:%v, id:%v, qid:%v", err, id, qid)
			continue
		}
	}

	return nil
}

func GetTransStatusWithInfo(qid int64) (status int, doc *api.XngResourceInfoDoc, err error) {
	//是否正在转码
	//for i := 0; i < api.ReTryTimes; i++ {
	isTrans, err := videoRedis.IsTransCoding(qid)
	if err != nil {
		xlog.Error("GetTransStatusWithInfo judge resource is traning error:%v, qid:%v", err, qid)
		return
	}
	if isTrans {
		status = TransStatusSubmit
		return
	}

	//是否曾经完成转码
	doc, err = GetResInfoByQid(qid)
	if err != nil {
		xlog.Error("GetTransStatusWithInfo get resource info error:%v, qid:%v", err, qid)
		return
	}
	if doc == nil {
		status = TransStatusNot
		return
	}
	if doc.TransCode == nil {
		status = TransStatusOldOver
		return
	}
	if *doc.TransCode == ResStatusMade || *doc.TransCode == ResStatusTrans {
		status = TransStatusOver
		return
	}
	status = TransStatusNot
	return
}

func SubmitTransCode(req *api.TransCodeReq, ty int) (status int, err error) {
	model := ResourceFactory(ty)
	if model == nil {
		err = errors.New("create transcodeModel error")
		return
	}
	status, err = model.TransCodeResource(req)
	if err != nil {
		return
	}
	return
}

func PushFailMessage(aid int64) (err error) {
	failItem := api.AlbumFailItems{
		ID:    aid,
		Errno: MakeAlbumFailErrorNo,
	}
	for i := 0; i < api.ReTryTimes; i++ {
		err = videoRedis.AddPushMessageForFail(failItem)
		if err == nil {
			break
		}
	}
	if err != nil {
		//conf.Logger.Error("push fail message to client error; qid : %v, item: %v, error: %v", qid, failItem, err)
		return err
	}
	//conf.Logger.Info("push fail message to client, item=%v", failItem)
	return nil
}

func PushSuccessMessage(doc *api.XngResourceInfoDoc, aid int64) error {
	item := api.AlbumSuccessItems{
		ID:       aid,
		TryTimes: api.PushMessageTrys,
		Size:     doc.Size,
		Du:       doc.Du,
		VW:       doc.W,
		VH:       doc.H,
	}
	//todo::json一下item
	var err error
	for i := 0; i < api.ReTryTimes; i++ {
		err = videoRedis.AddPushMessage(item)
		if err == nil {
			break
		}
	}
	if err != nil {
		xlog.Error("push success message to client error; qid: %v, item: %v, error: %v", doc.ResId, item, err)
		return err
	}
	return nil
}

func TryCopyResToAlbum(qid int64) error {

	//ch := make(chan *StsInfo, 1)
	//go func() {
	//	stsInfo, err := GetStsForMts()
	//	if err != nil {
	//		conf.Logger.Error("get sts information error", "stsInfo", stsInfo, "error", err)
	//		//return err
	//		ch <- nil
	//	}
	//	ch <- &stsInfo
	//}()
	//todo:这块逻辑提出去比较好
	doc, err := DaoByQid.GetDocByQid(qid)
	if err != nil {
		xlog.Error("get doc by qid error:%v, qid:%v", err, qid)
		return err
	}
	if doc == nil || doc.TransCode == nil {
		return errors.New("copy resource to album bucket error, qdoc not exists")
	}

	if *doc.TransCode == ResStatusNot {
		return errors.New("res status not trans")
	} else if *doc.TransCode == ResStatusMade {
		return nil
	}

	qry := bson.M{"_id": qid}
	updata := bson.M{"$set": bson.M{"trans": ResStatusMade}}
	err = DaoByQid.UpdateResourceDoc(qid, qry, updata)
	if err != nil {
		return err
	}
	keyName := strconv.FormatInt(qid, 10)
	stsInfo, err := GetStsForMts()
	if stsInfo == nil || err != nil {
		return errors.New(fmt.Sprintf("sts info is nil, err:%v", err))
	}
	err = CopyVideoResToBucket(keyName, conf.C.Bucket.Resource, conf.C.Bucket.Album, stsInfo, doc.Size) //改从资源桶进行拷贝
	if err != nil {
		xlog.Error("copy resource to album bucket error:%v, fileName:%v", err, keyName)
		return err
	}
	go func() {
		err = SetMetaContentType(keyName, conf.C.Bucket.Album, stsInfo, api.ContentTypeVideo)
		if err != nil {
			xlog.Error("set video content-type error, fileName=%v, error=%v", keyName, err)
			//return err
		}
		err = DelResFromBucket(keyName, stsInfo.Data.OutputBucket, stsInfo)
		if err != nil {
			xlog.Error("failed to delete temp resource : %v after trans, bucket : %v", keyName, stsInfo.Data.OutputBucket)
			//return err
		}
	}()

	return nil
}

//func HandleUserResDoc(doc *api.XngResourceInfoDoc, req api.CheckStatusReq) error {
//	mt := time.Now().UnixNano() / 1e6
//	upData := bson.M{"$set": bson.M{"size": doc.Size, "fmt": doc.Fmt, "mt": mt, "w": doc.W, "h": doc.H, "code": doc.Code}}
//	qry := bson.M{"_id": req.ResId, "qid": req.Qid}
//	mid, nerr := videoRedis.GetUserResUpdateRecord(req.ResId)
//	if nerr != nil {
//		conf.Logger.Error("failed to get user resource update record, resId=%v, mid=%v, error=%v", req.ResId, mid, nerr)
//		return nerr
//	}
//	nerr = DaoByMid.UpdateUserResourceDoc(qry, upData, mid)
//	if nerr != nil {
//		conf.Logger.Error("failed to update user resource doc, qryData=%v, resId=%v, mid=%v, error=%v", qry, req.ResId, mid, nerr)
//		return nerr
//	}
//	nerr = videoRedis.DelUserResUpdateRecord(req.ResId)
//	if nerr != nil {
//		conf.Logger.Error("failed to delete user resource update record, resId=%v, mid=%v, error=%v", req.ResId, mid, nerr)
//		return nerr
//	}
//	conf.Logger.Info("update user resource doc success, updata=%v", upData)
//	return nil
//}

func GetTransListKey(qid int64) string {
	key := fmt.Sprintf("%s%d", TransListBaseStr, qid)
	return key
}

//检查转码状态函数：根据转码状态进行
//未进行转码：通知用户失败
//转码中：将aid加入list, 添加当前用户的结果通知
//转码完成：通知用户制作成功
func HandleTransCompleted(req *api.CheckStatusReq) (err error) {
	//获取转码状态
	status, doc, err := GetTransStatusWithInfo(req.Qid)
	if err != nil {
		xlog.Error("get resource trans status error:%v, qid:%v", err, req.Qid)
		return
	}
	switch status {
	case TransStatusNot: //未转码，提交动作再转码之后，此时仍旧未转码，证明曾经转码失败
		err = PushFailMessage(req.Aid)
		if err != nil {
			xlog.Error("push fail message to client error:%v, req:%v", err, *req)
			return
		}
		xlog.Info("succ push fail message to client, req:%v", req)
	case TransStatusSubmit: //转码中：转码还未完成，完成后统一通知相关用户
		listKey := GetTransListKey(req.Qid)
		for i := 0; i < api.ReTryTimes; i++ {
			err = videoRedis.AddAlbumToTransList(listKey, req.Aid)
			if err == nil {
				break
			}
		}
		if err != nil {
			xlog.Error("add album record to trans list error:%v", err)
			return
		}
	case TransStatusOver: //转码完成：直接通知用户成功
		//stsInfo, err := GetStsForMts()
		//if err != nil {
		//	conf.Logger.Error("get sts information error, stsInfo: %v, error: %v ", stsInfo, err)
		//	return err
		//}
		if doc == nil {
			xlog.Error("get doc is nil but trans ststus is over, req:%v", req)
			err = errors.New("get doc is nil but trans ststus is over")
			return
		}
		err = TryCopyResToAlbum(req.Qid)
		if err != nil {
			xlog.Error("try to copy resource to album bucket error:%v", err)
			return
		}

		err = PushSuccessMessage(doc, req.Aid)
		if err != nil {
			xlog.Error("push success message to client error, qid:%v, aid:%v", req.Qid, req.Aid)
			return
		}
		xlog.Info("push success message to client, req:%v, doc:%v", req, doc)
	case TransStatusOldOver:
		if doc == nil {
			xlog.Error("get doc is nil but trans ststus is over, req:%v", req)
			err = errors.New("get doc is nil but trans ststus is over")
			return
		}
		keyName := strconv.FormatInt(req.Qid, 10)
		stsInfo, nerr := GetStsForMts()
		if stsInfo == nil || nerr != nil {
			return errors.New(fmt.Sprintf("sts info is nil, err:%v", err))
		}
		err = CopyLittleResToBucket(keyName, conf.C.Bucket.Resource, conf.C.Bucket.Album, stsInfo)
		if err != nil {
			return
		}
		qry := bson.M{"_id": req.Qid}
		mt := time.Now().UnixNano() / 1e6
		upData := bson.M{"$set": bson.M{"trans": ResStatusMade, "mt": mt}}
		err = DaoByQid.UpdateResourceDoc(req.Qid, qry, upData)
		if err != nil {
			xlog.Error("failed to update resource trans status, qid=%v, err=%v", req.Qid, err)
			return err
		}
		err = PushSuccessMessage(doc, req.Aid)
		if err != nil {
			xlog.Error("push success message to client error, qid:%v, aid:%v", req.Qid, req.Aid)
			return
		}
		xlog.Info("push success message to client for old resource, req:%v, doc:%v", req, doc)
	default:
		return errors.New("unKnow status type, status")
	}
	return
}

func GetResDocByQeTag(qeTag string) (qDoc *api.XngResourceInfoDoc, err error) {
	doc, err := DaoByTag.GetDocByTag(qeTag)
	if err != nil {
		return
	}
	if doc != nil {
		//获取qid对应的资源信息
		qDoc, err = DaoByQid.GetDocByQid(doc.Qid)
		if err != nil {
			qDoc = nil
			return
		}
		return
	}
	return
}

//暂时不方便用"提交转码"的接口，拷贝前面的代码过来，先临时给客户端用，如果以后有更好的方式再做修改
func AppUploadCallback(qid int64) error {
	var doc *api.XngResourceInfoDoc
	keyName := strconv.FormatInt(qid, 10)
	stsInfo, err := GetStsForMts()
	if stsInfo == nil || err != nil {
		return errors.New(fmt.Sprintf("sts info is nil, err:%v", err))
	}
	doc, err = DaoByQid.GetDocByQid(qid)
	if err != nil {
		xlog.Error("get video information error:%v, filename:%v", err, keyName)
		err = errors.New(fmt.Sprintf("callbakc get video info err:%v or doc is nil", err))
		return err
	}

	errIgnore := CopyVideoResToBucket(keyName, stsInfo.Data.InputBucket, conf.C.Bucket.Resource, stsInfo, doc.Size)
	if errIgnore != nil {
		xlog.Error("copy resource to resource bucket error:%v, fileName:%v", errIgnore, keyName)
		return errIgnore
	}
	errIgnore = SetMetaContentType(keyName, conf.C.Bucket.Resource, stsInfo, api.ContentTypeVideo)
	if errIgnore != nil {
		xlog.Error("set resource content-type error:%v, fileName:%v, type:%v, bucket:%v", errIgnore, keyName, api.ContentTypeVideo, conf.C.Bucket.Resource)
	}
	return nil
}
