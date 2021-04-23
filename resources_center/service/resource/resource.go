package resource

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/mts"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"hash"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo/bson"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/ids_api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByQid"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByTag"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/callbackMQ"
	resourceDao "xgit.xiaoniangao.cn/xngo/service/resources_center/dao/resource"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/sts"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils/alioss"
)

type CallbackParam struct {
	CallbackUrl      string `json:"callbackUrl"`
	CallbackBody     string `json:"callbackBody"`
	CallbackBodyType string `json:"callbackBodyType"`
}

type ConfigStruct struct {
	Expiration string     `json:"expiration"`
	Conditions [][]string `json:"conditions"`
}

var ResStatusNot = 0
var ResStatusTrans = 1
var ResStatusMade = 2

const (
	UploadStatusNot       = 0
	UploadStatusSuccess   = 2
	UploadStatusInterrupt = 3
)
const (
	ResourcesCodeH264          = "h264"
	ResourcesCodeH264WithPoint = "h.264"
	VideoCoverType             = "img"
)
const (
	MultiUploadChunkSize = 16777216
)
const (
	MaxMediaQueryTimes = 100
)

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

func GetUploadStatus(ctx context.Context, qeTag string) (status int, doc *api.XngResourceInfoDoc, err error) {
	redisExists, err := resourceDao.ExitMultiUploadRecord(qeTag)
	if err != nil {
		return
	}
	if redisExists == true {
		status = UploadStatusInterrupt
		return
	}
	qid, err := ByQeTag(ctx, qeTag)
	if err != nil {
		xlog.ErrorC(ctx, "failed to get qid by qetag, err:%v", err)
		return
	}
	if qid == "" {
		status = UploadStatusNot
		return
	}
	doc, err = ByID(ctx, qid)
	if err != nil {
		xlog.ErrorC(ctx, "failed to get media info by qid, err:%v", err)
		return
	}
	if doc != nil {
		if doc.Type == 10 && (doc.Cover >= 0 || doc.Fmt == "mov,mp4,m4a,3gp,3g2,mj2," || doc.W == 0 || doc.H == 0) { //排除xbd type为10得资源得干扰
			status = UploadStatusNot
			return
		}
		status = UploadStatusSuccess
		return
	}
	/*
		//资源是否已经存在，已上传
		isExists, err := ExistResByQeTag(qeTag)
		if err != nil {
			return
		}
		if isExists == true {
			status = UploadStatusSuccess

			return
		}
	*/
	status = UploadStatusNot
	return
}
func GetCallBackBase64(ctx context.Context, uploadType int) (callbackBase64 string, err error) {
	callback := &CallbackParam{}
	switch uploadType {
	case api.ResourceTypeImg:
		callback.CallbackBody = api.UploadImgCallBackBody
		//callback.CallbackBody = "{\"bucket\":${bucket},\"my_var\":${x:my_var}}"
	default:
		callback.CallbackBody = api.UploadResCommonCallBackBody
	}

	callback.CallbackBodyType = "application/json"
	//callback.CallbackUrl = api.UploadCallBackUrl
	callback.CallbackUrl = conf.C.CallBackUrl
	callbackStr, err := json.Marshal(callback)
	if err != nil {
		xlog.ErrorC(ctx, "GetUploadConfInfo error, callbackParam:%v, error:%v", callback, err)
		return
	}
	callbackBase64 = base64.StdEncoding.EncodeToString(callbackStr)
	return
}

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
			xlog.Error("get new distributed id error, resp data nil, resp:%v", resp.Data)
			//ch <- 0
			id = 0
			continue
		}
		break
	}
	ch <- id
}

func getGmtIso8601(expireEnd int64) string {
	var tokenExpire = time.Unix(expireEnd, 0).Format("2006-01-02T15:04:05Z")
	return tokenExpire
}

func GetUploadPolicy(expire int64) (policy string, err error) {
	var tokenExpire = getGmtIso8601(expire)
	var config ConfigStruct
	config.Expiration = tokenExpire
	var condition []string
	condition = append(condition, "starts-with")
	condition = append(condition, "$key")
	condition = append(condition, api.UploadDir)

	config.Conditions = append(config.Conditions, condition)
	result, err := json.Marshal(config)
	if err != nil {
		return
	}
	policy = base64.StdEncoding.EncodeToString(result)
	return
}

func GetUploadSignature(policy string, secret string) (signature string, err error) {
	h := hmac.New(func() hash.Hash { return sha1.New() }, []byte(secret))
	_, err = io.WriteString(h, policy)
	if err != nil {
		return
	}
	signature = base64.StdEncoding.EncodeToString(h.Sum(nil))
	return
}

func GetUploadStsName(resType int) string {
	switch resType {
	case api.ResourceTypeAlbum:
		return api.StsForAlbumUpload
	case api.ResourceTypeLive:
		return api.StsForMtsLive
	case api.ResourceTypeGuideVideo:
		return api.StsForLiveGuideUpload
	case api.ResourceTypeAPPLog:
		return api.StsForXNGAppLog
	default:
		return api.StsForUserUpload
	}
}

func GetUploadConfInfo(ctx context.Context, uploadType int, noCallback bool) (uploadInfo api.UploadInfo, err error) {
	if noCallback == false {
		callbackBase64, nerr := GetCallBackBase64(ctx, uploadType)
		if nerr != nil {
			err = nerr
			return
		}
		uploadInfo.Callback = callbackBase64
	}
	ch := make(chan int64, 1)
	go GetDistributedId(ch)
	stsName := GetUploadStsName(uploadType)
	stsData, err := sts.GetUpToken(ctx, stsName)
	if err != nil {
		return
	}
	if stsData == nil {
		err = errors.New("uptoken data is nil")
		return
	}
	var expire int64
	now := time.Now().Unix()
	expire = now + int64(stsData.ExpireSec)
	policy, err := GetUploadPolicy(expire)
	if err != nil {
		return
	}
	signature, err := GetUploadSignature(policy, stsData.SecretKey)
	if err != nil {
		return
	}

	uploadInfo.AccessKey = stsData.AccessKey
	uploadInfo.SecurityToken = stsData.SecurityToken
	uploadInfo.Host = fmt.Sprintf("https://%s.%s", stsData.Bucket, stsData.Endpoint)
	uploadInfo.InternalHost = fmt.Sprintf("https://%s.%s", stsData.Bucket, stsData.EndpointInternal)
	uploadInfo.Policy = policy
	uploadInfo.Signature = signature
	uploadInfo.SuccessActionStatus = "200"
	uploadInfo.ExpireSec = stsData.ExpireSec

	qid := <-ch
	if qid <= 0 {
		err = errors.New("get new distributed id error")
		return
	}
	uploadInfo.Key = strconv.FormatInt(qid, 10)
	if uploadInfo.Key == "" {
		err = errors.New("strconv new qid to string error")
		return
	}

	return
}

func OrganizeCommonDoc(ossCallback alioss.OssCallBack, customVar *api.CallbackCustomParam) (qDoc *api.XngResourceInfoDoc, err error) {
	upt := utils.GetMilliTime()
	//mt := upt
	qid, err := strconv.ParseInt(ossCallback.Filename, 10, 64)
	if err != nil {
		return
	}
	qDoc = &api.XngResourceInfoDoc{
		ResId: qid,
		Type:  customVar.Kind,
		QeTag: customVar.QeTag,
		Size:  ossCallback.Size,
		Upt:   upt,
		//Ct:    upt,
		Src: customVar.Project,
		Fmt: ossCallback.Format,
		Ort: 1,
		W:   ossCallback.Width,
		H:   ossCallback.Height,
		Mt:  upt,
		Ref: 1,
	}
	return
}

type ImgInfoResp struct {
	FileSize    map[string]string `json:"FileSize"`
	Format      map[string]string `json:"Format"`
	ImgHeight   map[string]string `json:"ImageHeight"`
	ImgWidth    map[string]string `json:"ImageWidth"`
	Orientation map[string]string `json:"Orientation"`
}

func GetImgInfo(ctx context.Context, filename string, resType int) (doc *api.XngResourceInfoDoc, err error) {

	url := alioss.GetImageSignURL(ctx, filename, []alioss.ImageAction{&alioss.Info{}})
	client := &http.Client{}
	imgResp, err := client.Get(url)
	if err != nil {
		return
	}
	defer func() {
		if imgResp != nil && imgResp.Body != nil {
			_ = imgResp.Body.Close()
		}
	}()
	body, err := ioutil.ReadAll(imgResp.Body)
	resp := ImgInfoResp{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		xlog.ErrorC(ctx, "fail to unmarshal imageInfo, url:%s, resp:%v, err:%v", url, fmt.Sprintf("%v", imgResp.Body), err)
		return
	}
	resId, err := strconv.ParseInt(filename, 10, 64)
	if err != nil {
		return
	}
	size, err := strconv.ParseInt(resp.FileSize["value"], 10, 64)
	if err != nil {
		return
	}
	width, err := strconv.Atoi(resp.ImgWidth["value"])
	if err != nil {
		return
	}
	height, err := strconv.Atoi(resp.ImgHeight["value"])
	if err != nil {
		return
	}
	ort := 1
	if resp.Orientation["value"] != "" {
		ortValue, errIgnore := strconv.Atoi(resp.Orientation["value"])
		if errIgnore != nil {
			err = errIgnore
			return
		}
		ort = ortValue
	}
	upt := time.Now().UnixNano() / 1e6
	doc = &api.XngResourceInfoDoc{
		ResId: resId,
		Type:  resType,
		Size:  size,
		Upt:   upt,
		Mt:    upt,
		Fmt:   resp.Format["value"],
		Ort:   ort,
		W:     width,
		H:     height,
		Ref:   1,
	}
	xlog.InfoC(ctx, "get snap image info, resp:%v, doc:%v", resp, doc)
	return
}
func QueryMediaInfoAsync(ctx context.Context, client *mts.Client, filename string, resType int, jobId, proj string) (doc *api.XngResourceInfoDoc, err error) {
	request := mts.CreateQueryMediaInfoJobListRequest()
	request.Scheme = "https"
	request.MediaInfoJobIds = jobId
	response, errIgnore := client.QueryMediaInfoJobList(request)
	if errIgnore != nil {
		err = errIgnore
		return
	}
	upt := time.Now().UnixNano() / 1e6
	fSize, errIgnore := strconv.ParseInt(response.MediaInfoJobList.MediaInfoJob[0].Properties.FileSize, 10, 64)
	if errIgnore != nil {
		err = errIgnore
		return
	}
	width, errIgnore := strconv.Atoi(response.MediaInfoJobList.MediaInfoJob[0].Properties.Width)
	if errIgnore != nil {
		err = errIgnore
		return
	}
	height, errIgnore := strconv.Atoi(response.MediaInfoJobList.MediaInfoJob[0].Properties.Height)
	if errIgnore != nil {
		err = errIgnore
		return
	}
	resId, errIgnore := strconv.ParseInt(filename, 10, 64)
	if errIgnore != nil {
		err = errIgnore
		return
	}
	doc = &api.XngResourceInfoDoc{
		ResId: resId,
		Type:  resType,
		Size:  fSize,
		Upt:   upt,
		Fmt:   response.MediaInfoJobList.MediaInfoJob[0].Properties.Format.FormatName,
		Ort:   1,
		W:     width,
		H:     height,
		Mt:    upt,
		Src:   proj,
		Ref:   1,
	}
	if resType == api.ResourceTypeVideo || resType == api.ResourceTypeMusic || resType == api.ResourceTypeVoice || resType == api.ResourceTypeAlbum {
		duration, errIgnore := strconv.ParseFloat(response.MediaInfoJobList.MediaInfoJob[0].Properties.Duration, 64)
		if errIgnore != nil {
			err = errIgnore
			return
		}
		duration = duration * 1000
		doc.Du = duration
		if resType == api.ResourceTypeVideo { //上传完成的资源暂时默认已完成转码
			doc.TransCode = &ResStatusTrans
		} else if resType == api.ResourceTypeAlbum {
			doc.TransCode = &ResStatusMade
		}
		doc.Code = response.MediaInfoJobList.MediaInfoJob[0].Properties.Streams.VideoStreamList.VideoStream[0].CodecName
	}
	return
}
func GetMediaInfoAsync(ctx context.Context, client *mts.Client, bucket string, location string, filename, filepath string, kind int, prod int, proj string, userData string, jobType string) (err error) {
	objectName := filepath + filename
	var jobId, requestId string
	for i := 0; i < 3; i++ {
		request := mts.CreateSubmitMediaInfoJobRequest()
		request.Scheme = "https"
		request.Async = requests.NewBoolean(true)
		request.PipelineId = utils.GetMediaInfoPipe(conf.Env)
		request.Input = fmt.Sprintf("{\"Bucket\":\"%s\",\"Location\":\"%s\",\"Object\":\"%s\"}", bucket, location, objectName)
		var errIgnore error
		request.UserData, errIgnore = utils.SetMNSCallBackData(filename, filepath, kind, prod, proj, jobType, userData)
		if errIgnore != nil {
			err = errIgnore
			xlog.ErrorC(ctx, "GetMediaInfoAsync.SetMNSCallBackData failed, err:%v, key:%v", errIgnore, objectName)
			continue
		}
		response, errIgnore := client.SubmitMediaInfoJob(request)
		if errIgnore != nil {
			err = errIgnore
			xlog.ErrorC(ctx, "GetMediaInfoAsync.SubmitMediaInfoJob failed, err:%v, key:%v", errIgnore, objectName)
			continue
		}
		if response == nil {
			err = errors.New("GetMediaInfoAsync.SubmitMediaInfoJob failed, resp is nil")
			continue
		}
		if response.MediaInfoJob.State == "Fail" {
			err = errors.New(fmt.Sprintf("GetMediaInfoAsync.SubmitMediaInfoJob failed, resp:[%v], key:[%v]", *response, objectName))
			continue
		}
		err = nil
		jobId = response.MediaInfoJob.JobId
		requestId = response.RequestId
		break
	}
	xlog.DebugC(ctx, "GetMediaInfoAsync.submit mediaInfo, key:[%v], jobID:[%v],requestID:[%v]", filename, jobId, requestId)
	return
}
func GetAudioInfoUnAsync(ctx context.Context, client *mts.Client, bucket string, location string, filename, filepath string, resType int) (doc *api.XngResourceInfoDoc, ret bool, err error) {
	startTime := time.Now()
	var jobId string
	objectName := filepath + filename
	request := mts.CreateSubmitMediaInfoJobRequest()
	//request.ConnectTimeout = time.Second * 4
	request.Scheme = "https"
	request.Async = requests.NewBoolean(false)
	request.Input = fmt.Sprintf("{\"Bucket\":\"%s\",\"Location\":\"%s\",\"Object\":\"%s\"}", bucket, location, objectName)
	response, errIgnore := client.SubmitMediaInfoJob(request)
	if errIgnore != nil || response == nil {
		xlog.ErrorC(ctx, "submit audio info unasync job err, err:%v, key:%v", errIgnore, objectName)
		return
	}
	if response.MediaInfoJob.State == "Fail" {
		if response.MediaInfoJob.Code == "MediainfoTimeOut" {
			xlog.WarnC(ctx, "timeout retry, key:%s", objectName)
			response, errIgnore := client.SubmitMediaInfoJob(request)
			if errIgnore != nil || response == nil {
				xlog.ErrorC(ctx, "submit media info async job err, err:%v, key:%v", errIgnore, objectName)
				return
			}
			if response.MediaInfoJob.State == "Fail" {
				xlog.ErrorC(ctx, "submit media info async job State fail, resp:%v, key:%v", *response, objectName)
				return
			}
		} else {
			xlog.ErrorC(ctx, "submit media info async job State fail, resp:%v, key:%v", *response, objectName)
			return
		}
	}
	jobId = response.MediaInfoJob.JobId
	upt := time.Now().UnixNano() / 1e6
	fSize, errIgnore := strconv.ParseInt(response.MediaInfoJob.Properties.Format.Size, 10, 64)
	if errIgnore != nil {
		err = errIgnore
		return
	}
	resId, errIgnore := strconv.ParseInt(filename, 10, 64)
	if errIgnore != nil {
		err = errIgnore
		return
	}
	duration, errIgnore := strconv.ParseFloat(response.MediaInfoJob.Properties.Format.Duration, 64)
	if errIgnore != nil {
		err = errIgnore
		return
	}
	duration = duration * 1000
	doc = &api.XngResourceInfoDoc{
		ResId: resId,
		Type:  resType,
		Size:  fSize,
		Upt:   upt,
		Fmt:   response.MediaInfoJob.Properties.Format.FormatName,
		Ort:   1,
		Mt:    upt,
		Ref:   1,
		Du:    duration,
	}
	ret = true
	xlog.DebugC(ctx, "GetAudioInfoUnAsync use time:[%v], jobid:[%v]", time.Since(startTime), jobId)
	return
}
func GetMediaInfoUnAsync(ctx context.Context, client *mts.Client, bucket string, location string, filename, filepath string, resType int) (doc *api.XngResourceInfoDoc, ret bool, err error) {
	startTime := time.Now()
	var jobId string
	objectName := filepath + filename
	request := mts.CreateSubmitMediaInfoJobRequest()
	//request.ConnectTimeout = time.Second * 4
	request.Scheme = "https"
	request.Async = requests.NewBoolean(false)
	request.Input = fmt.Sprintf("{\"Bucket\":\"%s\",\"Location\":\"%s\",\"Object\":\"%s\"}", bucket, location, objectName)
	response, errIgnore := client.SubmitMediaInfoJob(request)
	if errIgnore != nil || response == nil {
		xlog.ErrorC(ctx, "submit media info async job err, err:%v, key:%v", errIgnore, objectName)
		return
	}

	if response.MediaInfoJob.State == "Fail" {
		if response.MediaInfoJob.Code == "MediainfoTimeOut" {
			xlog.WarnC(ctx, "timeout retry, key:%s", objectName)
			response, errIgnore := client.SubmitMediaInfoJob(request)
			if errIgnore != nil || response == nil {
				xlog.ErrorC(ctx, "submit media info async job err, err:%v, key:%v", errIgnore, objectName)
				return
			}
			if response.MediaInfoJob.State == "Fail" {
				xlog.ErrorC(ctx, "submit media info async job State fail, resp:%v, key:%v", *response, objectName)
				return
			}
		} else {
			xlog.ErrorC(ctx, "submit media info async job State fail, resp:%v, key:%v", *response, objectName)
			return
		}
	}
	jobId = response.MediaInfoJob.JobId
	upt := time.Now().UnixNano() / 1e6
	fSize, errIgnore := strconv.ParseInt(response.MediaInfoJob.Properties.FileSize, 10, 64)
	if errIgnore != nil {
		err = errIgnore
		return
	}
	width, errIgnore := strconv.Atoi(response.MediaInfoJob.Properties.Width)
	if errIgnore != nil {
		err = errIgnore
		return
	}
	height, errIgnore := strconv.Atoi(response.MediaInfoJob.Properties.Height)
	if errIgnore != nil {
		err = errIgnore
		return
	}
	resId, errIgnore := strconv.ParseInt(strings.Replace(filename, ".mp4", "", -1), 10, 64)
	if errIgnore != nil {
		err = errIgnore
		return
	}
	doc = &api.XngResourceInfoDoc{
		ResId: resId,
		Type:  resType,
		Size:  fSize,
		Upt:   upt,
		Fmt:   response.MediaInfoJob.Properties.Format.FormatName,
		Ort:   1,
		W:     width,
		H:     height,
		Mt:    upt,
		Ref:   1,
	}
	if resType == api.ResourceTypeImg {
		doc.Fmt = api.SnapShotType
	} else if resType == api.ResourceTypeVideo || resType == api.ResourceTypeMusic || resType == api.ResourceTypeVoice || resType == api.ResourceTypeAlbum || resType == api.ResourceTypeGuideVideo {
		duration, errIgnore := strconv.ParseFloat(response.MediaInfoJob.Properties.Duration, 64)
		if errIgnore != nil {
			err = errIgnore
			return
		}
		duration = duration * 1000
		doc.Du = duration
		//if doc.Size > api.TransLimitSize && resType == api.ResourceTypeVideo {
		//	doc.TransCode = &ResStatusNot
		//} else {
		//	doc.TransCode = &ResStatusMade
		//}
		if resType == api.ResourceTypeVideo { //上传完成的资源暂时默认已完成转码
			doc.TransCode = &ResStatusTrans
		} else if resType == api.ResourceTypeAlbum {
			doc.TransCode = &ResStatusMade
		}
		doc.Code = response.MediaInfoJob.Properties.Streams.VideoStreamList.VideoStream[0].CodecName
	}
	ret = true
	xlog.DebugC(ctx, "GetMediaInfoUnAsync use time:[%v], jobid:[%v]", time.Since(startTime), jobId)
	return
}
func GetMediaInfo(ctx context.Context, client *mts.Client, bucket string, location string, filename, filepath string, resType int) (doc *api.XngResourceInfoDoc, err error) {
	var jobId string
	objectName := filepath + filename
	start := time.Now()
	for i := 0; i < 3; i++ {
		request := mts.CreateSubmitMediaInfoJobRequest()
		request.Scheme = "https"
		request.Async = requests.NewBoolean(true)
		request.PipelineId = utils.GetMediaInfoPipe(conf.Env)
		request.Input = fmt.Sprintf("{\"Bucket\":\"%s\",\"Location\":\"%s\",\"Object\":\"%s\"}", bucket, location, objectName)
		response, errIgnore := client.SubmitMediaInfoJob(request)
		if errIgnore != nil {
			xlog.ErrorC(ctx, "submit media info async job err, err:%v, key:%v", errIgnore, objectName)
			continue
		}
		if response == nil {
			continue
		}
		if response.MediaInfoJob.State == "Fail" {
			xlog.ErrorC(ctx, "submit media info async job failed, resp:%v, key:%v", *response, objectName)
			continue
		}
		jobId = response.MediaInfoJob.JobId
		break
	}
	if jobId == "" {
		err = errors.New("submit media info job failed, jobId is nil")
		return
	}
	xlog.DebugC(ctx, "GetMediaInfo.SubmitMediaInfoJob key:[%s] use time:[%v]", filename, time.Since(start))
	startTime := time.Now()
	for i := 0; i < MaxMediaQueryTimes; i++ {
		request := mts.CreateQueryMediaInfoJobListRequest()
		request.Scheme = "https"
		request.MediaInfoJobIds = jobId
		response, errIgnore := client.QueryMediaInfoJobList(request)
		if errIgnore != nil {
			err = errIgnore
			return
		}
		if response.MediaInfoJobList.MediaInfoJob[0].State == "Fail" {
			err = errors.New(fmt.Sprintf("failed to get media info, msg:%v", response.MediaInfoJobList.MediaInfoJob[0].Message))
			return
		}
		if response.MediaInfoJobList.MediaInfoJob[0].State == "Success" {
			upt := time.Now().UnixNano() / 1e6
			fSize, errIgnore := strconv.ParseInt(response.MediaInfoJobList.MediaInfoJob[0].Properties.FileSize, 10, 64)
			if errIgnore != nil {
				err = errIgnore
				return
			}
			width, errIgnore := strconv.Atoi(response.MediaInfoJobList.MediaInfoJob[0].Properties.Width)
			if errIgnore != nil {
				err = errIgnore
				return
			}
			height, errIgnore := strconv.Atoi(response.MediaInfoJobList.MediaInfoJob[0].Properties.Height)
			if errIgnore != nil {
				err = errIgnore
				return
			}
			resId, errIgnore := strconv.ParseInt(filename, 10, 64)
			if errIgnore != nil {
				err = errIgnore
				return
			}
			doc = &api.XngResourceInfoDoc{
				ResId: resId,
				Type:  resType,
				Size:  fSize,
				Upt:   upt,
				Fmt:   response.MediaInfoJobList.MediaInfoJob[0].Properties.Format.FormatName,
				Ort:   1,
				W:     width,
				H:     height,
				Mt:    upt,
				Ref:   1,
			}
			if resType == api.ResourceTypeImg {
				doc.Fmt = api.SnapShotType
			} else if resType == api.ResourceTypeVideo || resType == api.ResourceTypeMusic || resType == api.ResourceTypeVoice || resType == api.ResourceTypeAlbum {
				duration, errIgnore := strconv.ParseFloat(response.MediaInfoJobList.MediaInfoJob[0].Properties.Duration, 64)
				if errIgnore != nil {
					err = errIgnore
					return
				}
				duration = duration * 1000
				doc.Du = duration
				//if doc.Size > api.TransLimitSize && resType == api.ResourceTypeVideo {
				//	doc.TransCode = &ResStatusNot
				//} else {
				//	doc.TransCode = &ResStatusMade
				//}
				if resType == api.ResourceTypeVideo { //上传完成的资源暂时默认已完成转码
					doc.TransCode = &ResStatusTrans
				} else if resType == api.ResourceTypeAlbum {
					doc.TransCode = &ResStatusMade
				}
				doc.Code = response.MediaInfoJobList.MediaInfoJob[0].Properties.Streams.VideoStreamList.VideoStream[0].CodecName
			}
			break
		}
		if time.Since(startTime).Seconds() > time.Second.Seconds()*4 {
			err = errors.New("failed to query media info job, media job out of time")
			return
		}
	}
	xlog.DebugC(ctx, "GetMediaInfo.QueryMediaInfoJobList key:[%s] use time:[%v]", filename, time.Since(startTime))
	return
}

func SetContentTypeForCommon(ctx context.Context, fileName string, stsInfo *sts.UploadToken, resType string) error {
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
		xlog.ErrorC(ctx, "file:%v set Content-Type:%v error, error:%v", fileName, resType, err)
		return err
	}
	return nil
}

func SubmitSnapShotSync(client *mts.Client, sts *sts.UploadToken, objectName string, targetImageName string, du float64, srcBucket, destBucket string, startTime int64) (err error) {
	location := sts.Endpoint[:len(sts.Endpoint)-13]
	request := mts.CreateSubmitSnapshotJobRequest()
	request.Scheme = "https"
	request.SnapshotConfig = fmt.Sprintf("{\"OutputFile\": {\"Bucket\": \"%s\",\"Location\":\"%s\",\"Object\": \"%s\"},\"Time\":\"%d\"}", destBucket, location, targetImageName, startTime)
	request.Input = fmt.Sprintf("{\"Bucket\":\"%s\", \"Location\": \"%s\",\"Object\":\"%s\" }", srcBucket, location, objectName)
	snapResp, err := client.SubmitSnapshotJob(request)
	if err != nil {
		return
	}
	if snapResp == nil {
		return errors.New("error, snap response is nil")
	}
	return
}

func SubmitSnapFrameForComm(sts *sts.UploadToken, objectName string, targetImageName string, du float64, srcBucket, destBucket string, startTime int64) (err error) {

	endPoint := sts.EndpointInternal
	if conf.Env != lib.PROD {
		endPoint = sts.Endpoint
	}
	client, err := oss.New(endPoint, sts.AccessKey, sts.SecretKey, oss.SecurityToken(sts.SecurityToken))
	if err != nil {
		return
	}
	bucket, err := client.Bucket(srcBucket)
	if err != nil {
		return
	}

	style := fmt.Sprintf("video/snapshot,t_%d,f_jpg,ar_auto", startTime)
	process := fmt.Sprintf("%s|sys/saveas,o_%v,b_%v", style, base64.URLEncoding.EncodeToString([]byte(targetImageName)), base64.URLEncoding.EncodeToString([]byte(destBucket)))
	resp, err := bucket.ProcessObject(objectName, process)
	if err != nil {
		err = errors.New(fmt.Sprintf("SubmitSnapShotFrame, err:%v, resp:%v", err, resp))
		return
	}
	return
}

func OrganizeVideoDoc(ctx context.Context, fileName, filepath, qeTag, project string, stsData *sts.UploadToken, resType int) (qDoc *api.XngResourceInfoDoc, err error) {
	endPoint := stsData.Endpoint
	regionID := endPoint[4 : len(endPoint)-13]
	client, err := mts.NewClientWithStsToken(regionID, stsData.AccessKey, stsData.SecretKey, stsData.SecurityToken)
	if err != nil {
		xlog.ErrorC(ctx, "OrganizeVideoDoc create new client with stsToken error:%v, fileName:%v", err, fileName)
		return
	}
	if conf.Env == lib.PROD {
		client.Domain = api.MtsVpcDomain
	}
	loc := endPoint[:len(endPoint)-13]
	//获取视频资源信息
	qDoc, err = GetMediaInfo(ctx, client, stsData.Bucket, loc, fileName, filepath, resType)
	if err != nil || qDoc == nil {
		xlog.ErrorC(ctx, "OrganizeVideoDoc get video information error:%v, fileName:%v, bucket:%v", err, fileName, stsData.Bucket)
		return
	}
	qDoc.QeTag = qeTag
	qDoc.Src = project
	return
}

/*
func OrganizeVideoDoc(ctx context.Context, ossCallback alioss.OssCallBack, customVar *api.CallbackCustomParam, stsData *sts.UploadToken) (qDoc *api.XngResourceInfoDoc, err error) {
	endPoint := stsData.Endpoint
	regionID := endPoint[4 : len(endPoint)-13]
	client, err := mts.NewClientWithStsToken(regionID, stsData.AccessKey, stsData.SecretKey, stsData.SecurityToken)
	if err != nil {
		xlog.ErrorC(ctx, "OrganizeVideoDoc create new client with stsToken error:%v, req:%v", err, ossCallback)
		return
	}
	if conf.Env == lib.PROD {
		client.Domain = api.MtsVpcDomain
	}
	loc := endPoint[:len(endPoint)-13]
	go func() {
		//设置视频资源的contentType
		errIgnore := SetContentTypeForCommon(ctx, ossCallback.Filename, stsData, api.ContentTypeVideo)
		if errIgnore != nil {
			xlog.ErrorC(ctx, "OrganizeVideoDoc set resource content-type error:%v, fileName:%v, type:%v, bucket:%v", errIgnore, ossCallback.Filename, api.ContentTypeVideo, stsData.Bucket)
		}
	}()
	//获取视频资源信息
	qDoc, err = GetMediaInfo(ctx, client, stsData.Bucket, loc, ossCallback.Filename, api.ResourceTypeVideo)
	if err != nil || qDoc == nil {
		xlog.ErrorC(ctx, "OrganizeVideoDoc get video information error:%v, req:%v, bucket:%v", err, ossCallback, stsData.Bucket)
		return
	}
	qDoc.QeTag = customVar.QeTag
	qDoc.Src = customVar.Project
	return
}
*/

func GetSnapShot(ctx context.Context, fileName string, stsData *sts.UploadToken, snapId int64, du float64, code, srcBucket, destBucket string, startTime float64) (err error) {
	targetImageName := strconv.FormatInt(snapId, 10)
	if du < startTime {
		startTime = api.SnapConfShortTime
	}
	err = SubmitSnapFrameForComm(stsData, fileName, targetImageName, du, srcBucket, destBucket, int64(startTime))
	if err != nil {
		return
	}
	//if code == ResourcesCodeH264 {
	//	err = SubmitSnapFrameForComm(stsData, fileName, targetImageName, du, srcBucket, destBucket, int64(startTime))
	//	if err != nil {
	//		return
	//	}
	//} else {
	//	endPoint := stsData.Endpoint
	//	regionID := endPoint[4 : len(endPoint)-13]
	//	client, errIgnore := mts.NewClientWithStsToken(regionID, stsData.AccessKey, stsData.SecretKey, stsData.SecurityToken)
	//	if errIgnore != nil {
	//		xlog.ErrorC(ctx, "GetSnapShot create new client with stsToken error:%v, fileName:%v", errIgnore, fileName)
	//		err = errIgnore
	//		return
	//	}
	//	if conf.Env == lib.PROD {
	//		client.Domain = api.MtsVpcDomain
	//	}
	//	errIgnore = SubmitSnapShotSync(client, stsData, fileName, targetImageName, du, srcBucket, destBucket, int64(startTime))
	//	if errIgnore != nil {
	//		err = errIgnore
	//		return
	//	}
	//}
	return
}

func OrganizeSnapShotDoc(ctx context.Context, project string, snapId int64) (snapDoc *api.XngResourceInfoDoc, err error) {
	targetImageName := strconv.FormatInt(snapId, 10)
	snapDoc, err = GetImgInfo(ctx, targetImageName, api.ResourceTypeImg)
	if err != nil {
		xlog.ErrorC(ctx, "OrganizeVideoDoc get snap image information error:%v, keyName:%v", err, targetImageName)
		return
	}
	if snapDoc == nil {
		err = errors.New("OrganizeVideoDoc get snap image information error, doc is nil")
		return
	}
	snapDoc.Src = project
	return
}

/*
func GetSnapShot(ctx context.Context, ossCallback alioss.OssCallBack, stsData *sts.UploadToken, snapId int64, du float64, code string) (err error) {
	targetImageName := strconv.FormatInt(snapId, 10)
	if code == ResourcesCodeH264 {
		err = SubmitSnapFrameForComm(stsData, ossCallback.Filename, targetImageName, du)
		if err != nil {
			return
		}
	} else {
		endPoint := stsData.Endpoint
		regionID := endPoint[4 : len(endPoint)-13]
		client, errIgnore := mts.NewClientWithStsToken(regionID, stsData.AccessKey, stsData.SecretKey, stsData.SecurityToken)
		if errIgnore != nil {
			xlog.ErrorC(ctx, "GetSnapShot create new client with stsToken error:%v, req:%v", errIgnore, ossCallback)
			err = errIgnore
			return
		}
		if conf.Env == lib.PROD {
			client.Domain = api.MtsVpcDomain
		}
		errIgnore = SubmitSnapShotSync(client, stsData, ossCallback.Filename, targetImageName, du)
		if errIgnore != nil {
			err = errIgnore
			return
		}
	}
	return
}


func OrganizeSnapShotDoc(ctx context.Context, ossCallback alioss.OssCallBack, customVar *api.CallbackCustomParam, snapId int64) (snapDoc *api.XngResourceInfoDoc, err error) {
	targetImageName := strconv.FormatInt(snapId, 10)
	snapDoc, err = GetImgInfo(ctx, targetImageName, api.ResourceTypeImg)
	if err != nil {
		xlog.ErrorC(ctx, "OrganizeVideoDoc get snap image information error:%v, req:%v", err, ossCallback)
		return
	}
	if snapDoc == nil {
		err = errors.New("OrganizeVideoDoc get snap image information error, doc is nil")
		return
	}
	snapDoc.Src = customVar.Project
	return
}
*/

//添加新资源的信息记录到xng_qiniu库的两个col内
func AddNewMediaInfoToDB(qDoc *api.XngResourceInfoDoc) (err error) {
	err = DaoByTag.AddXngTagDoc(qDoc.QeTag, qDoc.ResId)
	if err != nil {
		return
	}
	//目前依旧使用int64的qid进行分库分表
	err = DaoByQid.InsertResourceDoc(qDoc.ResId, qDoc)
	if err != nil {
		return
	}
	return
}
func GetMapByXngDoc(doc *api.XngResourceInfoDoc) map[string]interface{} {
	resMap := make(map[string]interface{})
	resMap["_id"] = doc.ResId
	resMap["ty"] = doc.Type
	resMap["size"] = doc.Size
	resMap["qetag"] = doc.QeTag
	resMap["upt"] = doc.Upt
	resMap["mt"] = doc.Mt
	resMap["ct"] = doc.Ct
	resMap["src"] = doc.Src
	resMap["fmt"] = doc.Fmt
	resMap["ort"] = doc.Ort
	resMap["w"] = doc.W
	resMap["h"] = doc.H
	resMap["ref"] = doc.Ref
	resMap["du"] = doc.Du
	resMap["cover"] = doc.Cover
	resMap["covertp"] = doc.CoverTp
	resMap["code"] = doc.Code
	resMap["music_name"] = doc.MusicName
	resMap["vrate"] = doc.VRate
	if doc.TransCode != nil {
		resMap["trans"] = doc.TransCode
	}
	return resMap
}

//添加新资源的信息记录到xng_qiniu库的两个col内
func UpInsertMediaInfoToDB(qDoc *api.XngResourceInfoDoc) (err error) {
	err = DaoByTag.AddXngTagDoc(qDoc.QeTag, qDoc.ResId)
	if err != nil {
		return
	}
	//目前依旧使用int64的qid进行分库分表

	err = DaoByQid.UpInsertResourceDoc(qDoc.ResId, GetMapByXngDoc(qDoc))
	if err != nil {
		return
	}
	return
}
func AddMediaInfoCache(ctx context.Context, qDoc *api.XngResourceInfoDoc, qidKey string) (err error) {
	err = resourceDao.SetQeTagCache(qDoc.QeTag, qidKey, api.CacheMiddleTime)
	if err != nil {
		return
	}
	err = resourceDao.SetCache(ctx, qidKey, qDoc, api.CacheMiddleTime)
	if err != nil {
		return
	}
	return
}

func GetResContentType(resType int) (contentType string) {
	switch resType {
	case api.ResourceTypeImg:
		return api.ContentTypeImg
	case api.ResourceTypeVideo, api.ResourceTypeAlbum, api.ResourceTypeLive:
		return api.ContentTypeVideo
	case api.ResourceTypeLyric:
		return api.ContentTypeLrc
	case api.ResourceTypeTxt:
		return api.ContentTypeTxt
	case api.ResourceTypeMusic:
		return api.ContentTypeMusic
	}
	return
}
func GetMtsStsName(resType int) string {
	switch resType {
	case api.ResourceTypeAlbum:
		return api.StsForMtsAlbum
	case api.ResourceTypeLive:
		return api.StsForMtsLive
	case api.ResourceTypeGuideVideo:
		return api.StsForMtsLiveGuide

	default:
		return api.StsForMtsMedia
	}
}
func SetOssCallBackContentType(ctx context.Context, client *oss.Client, fileName, bucketName, resType string) error {
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return err
	}
	err = bucket.SetObjectMeta(fileName, oss.ContentType(resType))
	if err != nil {
		xlog.ErrorC(ctx, "file:%v set Content-Type:%v error, error:%v", fileName, resType, err)
		return err
	}
	return nil
}
func GetOssCallbackVideoDocV2(ctx context.Context, client *mts.Client, stsData *sts.UploadToken, fileName, filePath, qeTag, project string, bucket string, resType int) (qDoc *api.XngResourceInfoDoc, ret bool, err error) {
	endPoint := stsData.Endpoint
	loc := endPoint[:len(endPoint)-13]
	qDoc, ret, err = GetMediaInfoUnAsync(ctx, client, bucket, loc, fileName, filePath, resType)
	if err != nil {
		xlog.ErrorC(ctx, "GetOssCallbackVideoDoc.GetMediaInfo get video information error:%v, fileName:%v, bucket:%v", err, fileName, bucket)
		return
	}
	if ret == false {
		return
	}
	if qDoc == nil {
		err = errors.New("GetOssCallbackVideoDoc.GetMediaInfo failed doc is nil")
		return
	}
	qDoc.QeTag = qeTag
	qDoc.Src = project
	return
}
func GetOssResAudioDoc(ctx context.Context, client *mts.Client, stsData *sts.UploadToken, fileName, filePath, qeTag, project string, bucket string, resType int) (qDoc *api.XngResourceInfoDoc, ret bool, err error) {
	endPoint := stsData.Endpoint
	loc := endPoint[:len(endPoint)-13]
	qDoc, ret, err = GetAudioInfoUnAsync(ctx, client, bucket, loc, fileName, filePath, resType)
	if err != nil {
		xlog.ErrorC(ctx, "GetOssResAudioDoc.GetAudioInfoUnAsync get video information error:%v, fileName:%v, bucket:%v", err, fileName, bucket)
		return
	}
	if ret == false {
		return
	}
	if qDoc == nil {
		err = errors.New("GetOssResAudioDoc.GetAudioInfoUnAsync failed doc is nil")
		return
	}
	qDoc.QeTag = qeTag
	qDoc.Src = project
	return
}
func GetOssCallbackVideoDoc(ctx context.Context, client *mts.Client, stsData *sts.UploadToken, fileName, filePath, qeTag, project string, bucket string, resType int) (qDoc *api.XngResourceInfoDoc, err error) {
	endPoint := stsData.Endpoint
	loc := endPoint[:len(endPoint)-13]
	qDoc, ret, err := GetMediaInfoUnAsync(ctx, client, bucket, loc, fileName, filePath, resType)
	if err != nil {
		xlog.ErrorC(ctx, "GetOssCallbackVideoDoc.GetMediaInfo get video information error:%v, fileName:%v, bucket:%v", err, fileName, bucket)
		return
	}
	if ret == false {
		return
	}
	if qDoc == nil {
		err = errors.New("GetOssCallbackVideoDoc.GetMediaInfo failed doc is nil")
		return
	}
	qDoc.QeTag = qeTag
	qDoc.Src = project
	return
}

func GetOssCallbackSnapShot(ctx context.Context, ossClient *oss.Client, mtsClient *mts.Client, fileName string, stsData *sts.UploadToken, snapId int64, du float64, code, srcBucket, destBucket string, startTime float64) (err error) {
	targetImageName := strconv.FormatInt(snapId, 10)
	if du < startTime {
		startTime = api.SnapConfShortTime
	}
	bucket, errIgnore := ossClient.Bucket(srcBucket)
	if errIgnore != nil {
		err = errIgnore
		return
	}
	style := fmt.Sprintf("video/snapshot,t_%d,f_jpg,ar_auto", int64(startTime))
	process := fmt.Sprintf("%s|sys/saveas,o_%v,b_%v", style, base64.URLEncoding.EncodeToString([]byte(targetImageName)), base64.URLEncoding.EncodeToString([]byte(destBucket)))
	resp, err := bucket.ProcessObject(fileName, process)
	if err != nil {
		return
	}
	conf.OssImgShotWithProcessCounter.WithLabelValues("OssSnapShot").Inc()
	xlog.DebugC(ctx, "GetOssCallbackSnapShot.ProcessObject resp:[%v]", resp)
	//err = SubmitSnapShotSync(mtsClient, stsData, fileName, targetImageName, du, srcBucket, destBucket, int64(startTime))
	//if err != nil {
	//	return
	//}
	//xlog.DebugC(ctx, "SubmitSnapShotSync success snap:[%s] for h265", fileName)
	return
}

func DealOssCallBackMsg(ctx context.Context, ossCallback alioss.OssCallBack, customVar *api.CallbackCustomParam) (qDoc *api.XngResourceInfoDoc, err error) {
	qDoc = &api.XngResourceInfoDoc{}
	idCh := make(chan int64, 1)
	if customVar.Kind == api.ResourceTypeAlbum || customVar.Kind == api.ResourceTypeVideo {
		go GetDistributedId(idCh)
	}
	start := time.Now()
	stsName := GetMtsStsName(customVar.Kind)
	/*
		stsData, err := sts.GetUpToken(ctx, stsName)
		if err != nil {
			xlog.ErrorC(ctx, "DealOssCallBackMsg get sts information err:%v, req:%v", err, ossCallback)
			return
		}
		if stsData == nil {
			err = errors.New("uptoken data is nil")
			return
		}
		endPoint := stsData.EndpointInternal
		if conf.Env != lib.PROD {
			endPoint = stsData.Endpoint
		}
		ossClient, err := oss.New(endPoint, stsData.AccessKey, stsData.SecretKey, oss.SecurityToken(stsData.SecurityToken))
		if err != nil {
			return
		}
		regionID := endPoint[4 : len(endPoint)-13]
		mtsClient, err := mts.NewClientWithStsToken(regionID, stsData.AccessKey, stsData.SecretKey, stsData.SecurityToken)
		if err != nil {
			xlog.ErrorC(ctx, "OrganizeVideoDoc create new client with stsToken error:%v", err)
			return
		}
		if conf.Env == lib.PROD {
			mtsClient.Domain = api.MtsVpcDomain
		}
	*/
	ossInfo, err := GetAliOssClient(ctx, stsName)
	if err != nil {
		return
	}
	mtsInfo, err := GetAliMtsClient(ctx, stsName)
	if err != nil {
		return
	}
	ossClient := ossInfo.Client
	mtsClient := mtsInfo.Client
	stsData := mtsInfo.Sts
	contentType := GetResContentType(customVar.Kind)
	//err = SetContentTypeForCommon(ctx, ossCallback.Filename, stsData, contentType)
	err = SetOssCallBackContentType(ctx, ossClient, ossCallback.Filename, stsData.Bucket, contentType)
	if err != nil {
		xlog.ErrorC(ctx, "set resource content-type error:%v, fileName:%v, type:%v, bucket:%v", err, ossCallback.Filename, api.ContentTypeVideo, stsData.Bucket)
		return
	}
	switch customVar.Kind {
	case api.ResourceTypeVideo, api.ResourceTypeAlbum:
		start = time.Now()
		//获取视频资源信息并整理doc
		qDoc, err = GetOssCallbackVideoDoc(ctx, mtsClient, stsData, ossCallback.Filename, "", customVar.QeTag, customVar.Project, stsData.Bucket, customVar.Kind)
		//qDoc, err = OrganizeVideoDoc(ctx, ossCallback.Filename, customVar.QeTag, customVar.Project, stsData, customVar.Kind)
		if err != nil {
			return
		}
		if qDoc == nil {
			err = errors.New("DealOssCallBackMsg failed to get qdoc, qdoc is nil")
			return
		}
		xlog.DebugC(ctx, "DealOssCallBackMsg.GetOssCallbackVideoDoc key:[%s] use time:[%s]", ossCallback.Filename, time.Since(start))
		//对视频进行截图
		snapId := <-idCh
		if snapId <= 0 {
			err = errors.New("DealOssCallBackMsg get snapId err")
			return
		}
		start = time.Now()
		srcBucket, destBucket := stsData.Bucket, conf.C.Bucket.Resource
		err = GetOssCallbackSnapShot(ctx, ossClient, mtsClient, ossCallback.Filename, stsData, snapId, qDoc.Du, qDoc.Code, srcBucket, destBucket, api.SnapConfTime)
		//err = GetSnapShot(ctx, ossCallback.Filename, stsData, snapId, qDoc.Du, qDoc.Code, srcBucket, destBucket, api.SnapConfTime)
		if err != nil {
			xlog.ErrorC(ctx, "GetSnapShot failed to get snap shot, err:%v, req:%v, code:%v", err, ossCallback, qDoc.Code)
			return
		}
		xlog.DebugC(ctx, "DealOssCallBackMsg.GetOssCallbackSnapShot key:[%s] use time:[%s]", ossCallback.Filename, time.Since(start))
		go func() { //截图信息并发处理
			//获取截图信息并整理doc
			snapDoc, errIgnore := OrganizeSnapShotDoc(ctx, customVar.Project, snapId)
			if errIgnore != nil {
				xlog.ErrorC(ctx, "OrganizeSnapShotDoc failed to get snap shot information, err:%v, req:%v, code:%v", errIgnore, ossCallback, qDoc.Code)
				return
			}
			errIgnore = DaoByQid.InsertResourceDoc(snapId, snapDoc)
			if errIgnore != nil {
				xlog.ErrorC(ctx, "InsertResourceDoc failed to insert to DB, err:%v, req:%v, code:%v", errIgnore, ossCallback, qDoc.Code)
				return
			}
		}()
		qDoc.Cover = snapId
		qDoc.CoverTp = VideoCoverType
	case api.ResourceTypeLyric, api.ResourceTypeTxt:
		qDoc, err = OrganizeCommonDoc(ossCallback, customVar)
		if err != nil {
			return
		}
	case api.ResourceTypeVoice, api.ResourceTypeMusic:
		var ret bool
		qDoc, ret, err = GetOssResAudioDoc(ctx, mtsClient, stsData, ossCallback.Filename, "", customVar.QeTag, customVar.Project, stsData.Bucket, customVar.Kind)
		if err != nil {
			return
		}
		if ret == false || qDoc == nil {
			err = errors.New("DealOssCallBackMsg.GetOssResAudioDoc failed")
			return
		}
		qDoc.MusicName = customVar.MusicName
	case api.ResourceTypeImg:
		qDoc, err = GetImgInfo(ctx, ossCallback.Filename, customVar.Kind)
		if err != nil {
			return
		}
		qDoc.QeTag = customVar.QeTag
		qDoc.Src = customVar.Project
	case api.ResourceTypeLive: //直播类资源上传，无封面截图
		start = time.Now()
		ind := strings.LastIndex(ossCallback.Filename, "/")
		resName := ossCallback.Filename[ind+1:]
		resPath := ""
		if ind != -1 {
			resPath = ossCallback.Filename[0 : ind+1]
		}
		//获取直播视频资源信息并整理doc
		qDoc, err = GetOssCallbackVideoDoc(ctx, mtsClient, stsData, resName, resPath, customVar.QeTag, customVar.Project, stsData.Bucket, customVar.Kind)
		//qDoc, err = OrganizeVideoDoc(ctx, ossCallback.Filename, customVar.QeTag, customVar.Project, stsData, customVar.Kind)
		if err != nil {
			return
		}
		if qDoc == nil {
			err = errors.New("DealOssCallBackMsg failed to get qdoc, qdoc is nil")
			return
		}
		xlog.DebugC(ctx, "DealOssCallBackMsg.GetOssCallbackVideoDoc key:[%s] use time:[%s]", ossCallback.Filename, time.Since(start))
	default:
		err = errors.New("DealOssCallBackMsg UnKnow Resource Type")
		return
	}
	//目前依旧使用int64的qid进行分库分表
	err = AddNewMediaInfoToDB(qDoc)
	if err != nil {
		xlog.ErrorC(ctx, "Add new mediaInfo to DB failed, err:%v, req:%v, qDoc:%v", err, ossCallback, qDoc)
		return
	}
	return
}

func SetUploadLocalCache(ctx context.Context, fileName string, doc *api.XngResourceInfoDoc) (err error) {
	err = resourceDao.SetQeTagCache(doc.QeTag, fileName, api.CacheShortTime)
	if err != nil {
		xlog.ErrorC(ctx, "Add Uploaded LocalCache to redis failed, err:%v, Doc:%v", err, doc)
		return
	}
	err = resourceDao.SetCache(ctx, fileName, doc, api.CacheShortTime)
	if err != nil {
		xlog.ErrorC(ctx, "set upload cache failed, err:%v, fileName:%v, Doc:%v", err, fileName, doc)
		return
	}
	return
}

//func ByQeTag(ctx context.Context, qeTag string) (doc *api.XngResourceInfoDoc, err error) {
//	key, err := resourceDao.GetQeTagCache(qeTag)
//	if err != nil {
//		xlog.ErrorC(ctx, "Get Uploaded LocalCache from redis failed, err:%v, qetag:%v", err, qeTag)
//		return
//	}
//	if key != "" {
//		doc, err = resourceDao.GetCache(ctx, key)
//		if err != nil {
//			xlog.ErrorC(ctx, "GetCache from redis failed, err:%v, key:%v", err, key)
//			return
//		}
//		if doc != nil {
//			xlog.DebugC(ctx, "Get Uploaded LocalCache from redis success, Doc:%v", doc)
//			return
//		}
//	}
//缓存中不存在从mongo中查询
//	doc, err = videoService.GetResDocByQeTag(qeTag)
//	if err != nil {
//		xlog.ErrorC(ctx, "failed to GetResDocByQeTag, qetag:%v, err:%v", qeTag, err)
//		return
//	}
//	if doc != nil {
//		qid := strconv.FormatInt(doc.ResId, 10)
//		err = resourceDao.SetQeTagCache(qeTag, qid, api.CacheMiddleTime)
//		if err != nil {
//			return
//		}
//		err = resourceDao.SetCache(ctx, qid, doc, api.CacheMiddleTime)
//		if err != nil {
//			return
//		}
//	}
//	return
//}

type UploadMsg struct {
	ResId     string  `json:"id"`
	Type      int     `json:"ty"`
	Size      int64   `json:"size"`
	QeTag     string  `json:"qetag"`
	Upt       int64   `json:"upt"`
	Src       string  `json:"src"`
	Fmt       string  `json:"fmt"`
	W         int     `json:"w,omitempty"`
	H         int     `json:"h,omitempty"`
	Du        float64 `json:"du,omitempty"`
	Cover     string  `json:"cover,omitempty"`
	Code      string  `json:"code,omitempty"`
	MyVar     string  `json:"my_var"`
	MusicName string  `json:"music_name,omitempty"` //音乐类资源的歌曲名字
}

func NotifyUploadMsg(ctx context.Context, product int, project string, res *api.XngResourceInfoDoc, myVar string) (err error) {
	topic, ok := api.MqTopicMap[product]
	if !ok {
		err = errors.New(fmt.Sprintf("unknown res type: %v", product))
		return
	}

	tag := fmt.Sprintf("src_%s", project)

	data := &UploadMsg{
		ResId:     fmt.Sprintf("%d", res.ResId),
		Type:      res.Type,
		Size:      res.Size,
		QeTag:     res.QeTag,
		Upt:       res.Upt,
		Src:       res.Src,
		Fmt:       res.Fmt,
		W:         res.W,
		H:         res.H,
		Du:        res.Du,
		Cover:     fmt.Sprintf("%d", res.Cover),
		Code:      res.Code,
		MusicName: res.MusicName,
		MyVar:     myVar,
	}

	err = callbackMQ.NotifyByMq(ctx, topic, tag, data)
	if err != nil {
		return
	}
	return
}
func GetHeaderSignature(method, AccSecret string, md5Value, contentType, Date string, ossHeaders, Resource string) (signedStr string, err error) {
	if Date == "" || Resource == "" {
		err = errors.New("signature failed Data or CanonicalizedResource is nil")
		return
	}
	var signatureStr = []string{method, "\n", md5Value, "\n", contentType, "\n", Date, "\n", ossHeaders, Resource}

	signature := strings.Join(signatureStr, "")
	h := hmac.New(func() hash.Hash { return sha1.New() }, []byte(AccSecret))
	_, err = io.WriteString(h, signature)
	if err != nil {
		return
	}
	signedStr = base64.StdEncoding.EncodeToString(h.Sum(nil))
	return
}

func GetAuthorization(AccKey, signature string) string {
	return "OSS " + AccKey + ":" + signature
}

func GetStaticUploadConf(ctx context.Context, ObjectKey string, stsData *sts.UploadToken) (resp *api.RespStaticUploadConf) {
	url := fmt.Sprintf("https://%s.%s/%s", stsData.Bucket, stsData.Endpoint, ObjectKey)
	urlInternal := fmt.Sprintf("https://%s.%s/%s", stsData.Bucket, stsData.EndpointInternal, ObjectKey)
	host := fmt.Sprintf("%s.%s", stsData.Bucket, stsData.Endpoint)
	hostInternal := fmt.Sprintf("%s.%s", stsData.Bucket, stsData.EndpointInternal)
	date := time.Now().UTC().Format(http.TimeFormat)
	resp = &api.RespStaticUploadConf{
		Method:        http.MethodPut,
		Url:           url,
		UrlInternal:   urlInternal,
		Host:          host,
		HostInternal:  hostInternal,
		Date:          date,
		SecurityToken: stsData.SecurityToken,
		ExpireSec:     stsData.ExpireSec,
	}
	return
}

func GetNewChunkInfo(fileSize int64) (chunks []api.ChunkData, chunkCnt int) {
	chunkNum := fileSize/MultiUploadChunkSize + 1
	chunkCnt = int(chunkNum)
	lastChunkSize := fileSize % MultiUploadChunkSize
	chunks = make([]api.ChunkData, chunkCnt)
	for i := 0; i < chunkCnt; i++ {
		chunks[i].Number = i + 1
		chunks[i].Offset = int64((i) * MultiUploadChunkSize)
		if i+1 == chunkCnt {
			chunks[i].Size = lastChunkSize
		} else {
			chunks[i].Size = MultiUploadChunkSize
		}
		chunks[i].Ready = false
	}
	return
}

func InitMultiUpload(ctx context.Context, objectName string, client *oss.Client, bucketName string) (uploadID string, err error) {
	/*
		endpoint := stsData.Endpoint
		if conf.Env == lib.PROD {
			endpoint = stsData.EndpointInternal
		}
		client, err := oss.New(endpoint, stsData.AccessKey, stsData.SecretKey, oss.SecurityToken(stsData.SecurityToken))
		if err != nil {
			xlog.ErrorC(ctx, "init oss client err, err:%v", err)
			return
		}
	*/
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		xlog.ErrorC(ctx, "init oss bucket err, err:%v", err)
		return
	}
	storageType := oss.ObjectStorageClass(oss.StorageStandard)

	imur, err := bucket.InitiateMultipartUpload(objectName, storageType)
	if err != nil {
		xlog.ErrorC(ctx, "init multiPart upload err, err:%v, objectName:%v", err, objectName)
		return
	}
	uploadID = imur.UploadID
	return
}

func AddMultiRecord(qeTag, uploadID, key string, size int64) (err error) {
	uploadDoc := &api.MultiUploadRecord{
		Key:      key,
		UploadID: uploadID,
		Size:     size,
	}
	err = resourceDao.AddMultiUploadRecord(qeTag, uploadDoc)
	if err != nil {
		return
	}
	abortDoc := &api.MultiAbortRecord{
		Key:      key,
		UploadID: uploadID,
		QeTag:    qeTag,
	}

	err = resourceDao.AddMultiAbortRecord(abortDoc)
	if err != nil {
		return
	}
	return
}

func GetResumeConfigs(ctx context.Context, key string, uploadID string, fileSize int64, client *oss.Client, bucketName string) (parts []api.PartData, chunks []api.ChunkData, chunkCnt int, err error) {
	chunks, chunkCnt = GetNewChunkInfo(fileSize)
	imur := oss.InitiateMultipartUploadResult{
		XMLName:  xml.Name{},
		Bucket:   bucketName,
		Key:      key,
		UploadID: uploadID,
	}
	/*
		endpoint := stsData.Endpoint
		if conf.Env == lib.PROD {
			endpoint = stsData.EndpointInternal
		}
		client, err := oss.New(endpoint, stsData.AccessKey, stsData.SecretKey, oss.SecurityToken(stsData.SecurityToken))
		if err != nil {
			xlog.ErrorC(ctx, "GetResumeConfigs init oss client err, err:%v", err)
			return
		}
	*/
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		xlog.ErrorC(ctx, "GetResumeConfigs init oss bucket err, err:%v", err)
		return
	}
	var part api.PartData
	var option oss.Option
	startTime := time.Now()
	for i := 0; i < 10; i++ { //最大10次，oss支持每次最大返回1000，最大共获取10000个part数据
		lsRes, errIgnore := bucket.ListUploadedParts(imur, option)
		if errIgnore != nil {
			err = errIgnore
			xlog.ErrorC(ctx, "GetResumeConfigs ListUploadedParts err, err:%v, use time:%v", errIgnore, time.Since(startTime))
			return
		}
		if lsRes.UploadedParts == nil {
			break
		}
		nextMarkerNum, errIgnore := strconv.Atoi(lsRes.NextPartNumberMarker)
		if errIgnore != nil {
			err = errIgnore
			return
		}
		for _, data := range lsRes.UploadedParts {
			part.Etag = data.ETag
			part.Num = data.PartNumber
			if part.Num > chunkCnt {
				err = errors.New("part num out of chunks index")
				return
			}
			chunks[part.Num-1].Ready = true
			parts = append(parts, part)
		}
		if lsRes.IsTruncated == true { //未获取全部part，继续获取
			option = oss.PartNumberMarker(nextMarkerNum) //下次获取得起始位置
			continue
		}
		break
	}
	return
}

func GetParts(datas []api.PartData) (parts []oss.UploadPart) {
	for _, data := range datas {
		part := oss.UploadPart{
			XMLName:    xml.Name{},
			PartNumber: data.Num,
			ETag:       data.Etag,
		}
		parts = append(parts, part)
	}
	return
}

func MergeMultiParts(ctx context.Context, req *api.ReqCheckMultiUpload, stsData *sts.UploadToken, client *oss.Client) (ret bool, err error) {
	startTime := time.Now()
	imur := oss.InitiateMultipartUploadResult{
		XMLName:  xml.Name{},
		Bucket:   stsData.Bucket,
		Key:      req.Key,
		UploadID: req.UploadID,
	}
	parts := GetParts(req.Parts)
	/*
		endpoint := stsData.EndpointInternal
		if conf.Env != lib.PROD {
			endpoint = stsData.Endpoint
		}
		client, err := oss.New(endpoint, stsData.AccessKey, stsData.SecretKey, oss.SecurityToken(stsData.SecurityToken))
		if err != nil {
			xlog.ErrorC(ctx, "init oss client err, err:%v", err)
			return
		}
	*/
	bucket, err := client.Bucket(stsData.Bucket)
	if err != nil {
		xlog.ErrorC(ctx, "init oss bucket err, err:%v", err)
		return
	}
	_, nerr := bucket.CompleteMultipartUpload(imur, parts)
	if nerr != nil {
		xlog.InfoC(ctx, "failed to complete multiUpload, reason:%v", nerr)
		ret = false
		return
	}
	ret = true
	xlog.DebugC(ctx, "CompleteMultiUpload use time : %v", time.Since(startTime))
	return
}

func GetMediaBaseInfo(ctx context.Context, client *mts.Client, loc string, bucket string, filename string, resType int) (doc *api.XngResourceInfoDoc, err error) {
	/*
		endPoint := stsData.Endpoint
		regionID := endPoint[4 : len(endPoint)-13]
		client, nerr := mts.NewClientWithStsToken(regionID, stsData.AccessKey, stsData.SecretKey, stsData.SecurityToken)
		if nerr != nil {
			err = nerr
			return
		}
		if conf.Env == lib.PROD {
			client.Domain = api.MtsVpcDomain
		}
		location := endPoint[:len(endPoint)-13]
	*/
	request := mts.CreateSubmitMediaInfoJobRequest()
	request.Scheme = "https"
	request.Async = requests.NewBoolean(false)
	request.Input = fmt.Sprintf("{\"Bucket\":\"%s\",\"Location\":\"%s\",\"Object\":\"%s\"}", bucket, loc, filename)
	response, err := client.SubmitMediaInfoJob(request)
	if err != nil {
		return
	}
	if response == nil {
		err = errors.New("submit media info job error, response nil")
		return
	}
	if response.MediaInfoJob.State == "Fail" {
		xlog.ErrorC(ctx, "submit media info job failed, resp:%v, key:%v", *response, filename)
		err = errors.New("submit media info job failed")
		return
	}
	upt := time.Now().UnixNano() / 1e6
	fSize, err := strconv.ParseInt(response.MediaInfoJob.Properties.FileSize, 10, 64)
	if err != nil {
		return
	}
	resId, err := strconv.ParseInt(filename, 10, 64)
	if err != nil {
		return
	}
	doc = &api.XngResourceInfoDoc{
		ResId: resId,
		Type:  resType,
		Size:  fSize,
		Upt:   upt,
		Fmt:   response.MediaInfoJob.Properties.Format.FormatName,
		Ort:   1,
		Mt:    upt,
		Ref:   1,
	}
	if resType == api.ResourceTypeMusic || resType == api.ResourceTypeVoice {
		duration, errIgnore := strconv.ParseFloat(response.MediaInfoJob.Properties.Duration, 64)
		if errIgnore != nil {
			err = errIgnore
			return
		}
		duration = duration * 1000
		doc.Du = duration
		doc.Code = response.MediaInfoJob.Properties.Streams.VideoStreamList.VideoStream[0].CodecName
	}
	return
}
func GetMultiBaseDoc(ctx context.Context, resKey, qetag, project string, kind, w, h int, du float64, size int64) (qDoc *api.XngResourceInfoDoc, err error) {
	resId, err := strconv.ParseInt(resKey, 10, 64)
	if err != nil {
		return
	}
	upt := time.Now().UnixNano() / 1e6
	qDoc = &api.XngResourceInfoDoc{
		ResId: resId,
		Type:  kind,
		Size:  size,
		QeTag: qetag,
		Upt:   upt,
		Mt:    upt,
		Ct:    upt,
		Src:   project,
		Ort:   1,
		W:     w,
		H:     h,
		Ref:   1,
		Du:    du,
	}
	if kind == api.ResourceTypeVideo { //上传完成的资源暂时默认已完成转码
		qDoc.TransCode = &ResStatusTrans
	} else if kind == api.ResourceTypeAlbum {
		qDoc.TransCode = &ResStatusMade
	}
	return
}
func GetMultiUplaodResInfo(ctx context.Context, req api.ReqCheckMultiUpload, stsData *sts.UploadToken, ossClient *oss.Client, mtsClient *mts.Client) (retDoc *api.XngResourceInfoDoc, ok bool, err error) {
	ok = true
	switch req.Kind {
	case api.ResourceTypeImg:
		resDoc, nerr := GetImgInfo(ctx, req.Key, api.ResourceTypeImg)
		if nerr != nil {
			err = nerr
			return
		}
		resDoc.QeTag = req.QeTag
		resDoc.Src = req.Project
		retDoc = resDoc
	case api.ResourceTypeVideo, api.ResourceTypeAlbum:
		idCh := make(chan int64, 1)
		go GetDistributedId(idCh)
		resDoc, ret, nerr := GetOssCallbackVideoDocV2(ctx, mtsClient, stsData, req.Key, "", req.QeTag, req.Project, stsData.Bucket, req.Kind)
		//resDoc, nerr := OrganizeVideoDoc(ctx, req.Key, "", req.QeTag, req.Project, stsData, req.Kind)
		if nerr != nil {
			err = nerr
			return
		}
		if ret == false {
			ok = ret
			return
		}
		snapId := <-idCh
		if snapId <= 0 {
			err = errors.New("GetMultiUplaodResInfoV2 failed to get snapId, id is nil")
			return
		}
		nerr = GetOssCallbackSnapShot(ctx, ossClient, mtsClient, req.Key, stsData, snapId, resDoc.Du, resDoc.Code, stsData.Bucket, conf.C.Bucket.Resource, api.SnapConfTime)
		//nerr = GetSnapShot(ctx, req.Key, stsData, snapId, resDoc.Du, resDoc.Code, stsData.Bucket, conf.C.Bucket.Resource, api.SnapConfTime)
		if nerr != nil {
			err = nerr
			return
		}
		//获取截图信息并整理doc
		snapDoc, nerr := OrganizeSnapShotDoc(ctx, req.Project, snapId)
		if nerr != nil {
			err = nerr
			return
		}
		nerr = DaoByQid.InsertResourceDoc(snapId, snapDoc)
		if nerr != nil {
			err = nerr
			return
		}
		resDoc.Cover = snapId
		resDoc.CoverTp = VideoCoverType
		retDoc = resDoc
	default: //处理非图片，视频类的资源，（目前为music, voice, text, lyric类资源）
		endPoint := stsData.EndpointInternal
		if conf.Env != lib.PROD {
			endPoint = stsData.Endpoint
		}
		loc := endPoint[:len(endPoint)-13]
		resDoc, nerr := GetMediaBaseInfo(ctx, mtsClient, loc, stsData.Bucket, req.Key, req.Kind)
		if nerr != nil {
			err = nerr
			return
		}
		resDoc.QeTag = req.QeTag
		resDoc.Src = req.Project
		retDoc = resDoc
	}
	return
}
func GetMultiUplaodResInfoV2(ctx context.Context, req api.ReqCheckMultiUpload, stsData *sts.UploadToken, ossClient *oss.Client, mtsClient *mts.Client, mediaInfo *api.MultiMediaInfo) (retDoc *api.XngResourceInfoDoc, err error) {
	switch req.Kind {
	case api.ResourceTypeImg:
		resDoc, nerr := GetImgInfo(ctx, req.Key, api.ResourceTypeImg)
		if nerr != nil {
			err = nerr
			return
		}
		resDoc.QeTag = req.QeTag
		resDoc.Src = req.Project
		retDoc = resDoc
	case api.ResourceTypeVideo, api.ResourceTypeAlbum:
		idCh := make(chan int64, 1)
		go GetDistributedId(idCh)
		resDoc, nerr := GetMultiBaseDoc(ctx, req.Key, req.QeTag, req.Project, req.Kind, mediaInfo.W, mediaInfo.H, mediaInfo.Du, mediaInfo.Size)
		if nerr != nil {
			err = nerr
			return
		}
		resDoc.Fmt = mediaInfo.Fmt
		resDoc.Code = mediaInfo.Code
		if mediaInfo.Code == "" || mediaInfo.Fmt == "" {
			endPoint := stsData.Endpoint
			loc := endPoint[:len(endPoint)-13]
			data := api.MultiVideoUserData{
				UserData: req.UserData,
				Qetag:    req.QeTag,
			}
			userByte, nerr := json.Marshal(data)
			if nerr != nil {
				err = nerr
				return
			}
			nerr = GetMediaInfoAsync(ctx, mtsClient, stsData.Bucket, loc, req.Key, "", req.Kind, req.Product, req.Project, string(userByte), api.MTSJobTypeVideoInfo) //异步提交作业获取全部信息
			if nerr != nil {
				err = nerr
				return
			}
		}
		snapId := <-idCh
		if snapId <= 0 {
			err = errors.New("GetMultiUplaodResInfoV2 failed to get snapId, id is nil")
			return
		}
		nerr = GetOssCallbackSnapShot(ctx, ossClient, mtsClient, req.Key, stsData, snapId, resDoc.Du, resDoc.Code, stsData.Bucket, conf.C.Bucket.Resource, api.SnapConfTime)
		if nerr != nil {
			err = nerr
			return
		}
		//获取截图信息并整理doc
		snapDoc, nerr := OrganizeSnapShotDoc(ctx, req.Project, snapId)
		if nerr != nil {
			err = nerr
			return
		}
		nerr = DaoByQid.InsertResourceDoc(snapId, snapDoc)
		if nerr != nil {
			err = nerr
			return
		}
		resDoc.Cover = snapId
		resDoc.CoverTp = VideoCoverType
		retDoc = resDoc
	default: //处理非图片，视频类的资源，（目前为music, voice, text, lyric类资源）
		endPoint := stsData.EndpointInternal
		if conf.Env != lib.PROD {
			endPoint = stsData.Endpoint
		}
		loc := endPoint[:len(endPoint)-13]
		resDoc, nerr := GetMediaBaseInfo(ctx, mtsClient, loc, stsData.Bucket, req.Key, req.Kind)
		if nerr != nil {
			err = nerr
			return
		}
		resDoc.QeTag = req.QeTag
		resDoc.Src = req.Project
		retDoc = resDoc
	}
	return
}
func DelMultiRecord(qeTag, key, uploadID string) (err error) {
	err = resourceDao.DelMultiUploadRecord(qeTag)
	if err != nil {
		return
	}
	abortDoc := &api.MultiAbortRecord{
		Key:      key,
		UploadID: uploadID,
		QeTag:    qeTag,
	}
	err = resourceDao.DelMultiAbortRecord(abortDoc)
	if err != nil {
		return
	}
	return
}

func GetEtagList(ctx context.Context, key, uploadID string, stsData *sts.UploadToken, client *oss.Client) (parts []api.PartData, err error) {
	imur := oss.InitiateMultipartUploadResult{
		XMLName:  xml.Name{},
		Bucket:   stsData.Bucket,
		Key:      key,
		UploadID: uploadID,
	}
	/*
		endpoint := stsData.Endpoint
		if conf.Env == lib.PROD {
			endpoint = stsData.EndpointInternal
		}
		client, err := oss.New(endpoint, stsData.AccessKey, stsData.SecretKey, oss.SecurityToken(stsData.SecurityToken))
		if err != nil {
			xlog.ErrorC(ctx, "GetEtagList init oss client err, err:%v", err)
			return
		}
	*/
	bucket, err := client.Bucket(stsData.Bucket)
	if err != nil {
		xlog.ErrorC(ctx, "GetEtagList init oss bucket err, err:%v", err)
		return
	}
	var part api.PartData
	var option oss.Option
	startTime := time.Now()
	for i := 0; i < 10; i++ { //最大10次，oss支持每次最大返回1000，最大共获取10000个part数据
		lsRes, errIgnore := bucket.ListUploadedParts(imur, option)
		if errIgnore != nil {
			err = errIgnore
			xlog.ErrorC(ctx, "GetEtagList.ListUploadedParts err, err:%v, use time:%v", errIgnore, time.Since(startTime))
			return
		}
		if lsRes.UploadedParts == nil {
			break
		}
		nextMarkerNum, errIgnore := strconv.Atoi(lsRes.NextPartNumberMarker)
		if errIgnore != nil {
			err = errIgnore
			return
		}
		for _, data := range lsRes.UploadedParts {
			part.Etag = data.ETag
			part.Num = data.PartNumber
			parts = append(parts, part)
		}
		if lsRes.IsTruncated == true { //未获取全部part，继续获取
			option = oss.PartNumberMarker(nextMarkerNum) //下次获取得起始位置
			continue
		}
		break
	}
	return
}

func GetAliOssClient(ctx context.Context, key string) (client *api.AliOssClient, err error) {
	if api.StsOssClient[key] != nil && api.StsOssClient[key].Client != nil {
		client = api.StsOssClient[key]
	} else {
		xlog.DebugC(ctx, "sts client:[%s] is nil, retry to get client.", key)
		stsData, errIgnore := sts.GetUpToken(ctx, key)
		if errIgnore != nil {
			err = errIgnore
			xlog.Error("GetAliOssClient.GetUpToken err:%v, key:%s", errIgnore, key)
			return
		}
		if stsData == nil {
			xlog.Error("GetAliOssClient get sts information err, sts data is nil.")
			return
		}
		endPoint := stsData.EndpointInternal
		if conf.Env != lib.PROD {
			endPoint = stsData.Endpoint
		}
		ossClient, errIgnore := oss.New(endPoint, stsData.AccessKey, stsData.SecretKey, oss.SecurityToken(stsData.SecurityToken))
		if errIgnore != nil {
			err = errIgnore
			xlog.Error("GetAliOssClient.oss.New err:%v, key:%s", errIgnore, key)
			return
		}
		client = &api.AliOssClient{
			Client: ossClient,
			Sts:    stsData,
		}
		api.StsOssClient[key] = client
	}
	return
}

func GetAliMtsClient(ctx context.Context, key string) (client *api.AliMtsClient, err error) {
	if api.StsMtsClient[key] != nil && api.StsMtsClient[key].Client != nil {
		client = api.StsMtsClient[key]
	} else {
		stsData, errIgnore := sts.GetUpToken(context.Background(), key)
		if errIgnore != nil {
			err = errIgnore
			xlog.Error("GetAliMtsClient.GetUpToken err:%v, key:%s", err, key)
			return
		}
		if stsData == nil {
			xlog.Error("GetAliMtsClient get sts information err, sts data is nil.")
			return
		}
		endPoint := stsData.EndpointInternal
		if conf.Env != lib.PROD {
			endPoint = stsData.Endpoint
		}
		regionID := endPoint[4 : len(endPoint)-13]
		mtsClient, errIgnore := mts.NewClientWithStsToken(regionID, stsData.AccessKey, stsData.SecretKey, stsData.SecurityToken)
		if errIgnore != nil {
			err = errIgnore
			xlog.Error("GetAliMtsClient.NewClientWithStsToken error:%v, key:%s", err, key)
			return
		}
		if conf.Env == lib.PROD {
			mtsClient.Domain = api.MtsVpcDomain
		}
		client = &api.AliMtsClient{
			Client: mtsClient,
			Sts:    stsData,
		}
		api.StsMtsClient[key] = client
	}
	return
}

//
//func GetMultiRecordSize(ctx context.Context, qetag string, kind int) (size int64, err error) {
//	if kind != api.ResourceTypeVideo && kind != api.ResourceTypeAlbum {
//		return
//	}
//	record, err := resourceDao.GetMultiUploadRecord(qetag)
//	if err != nil {
//		return
//	}
//	if record == nil || record.Size == 0 {
//		err = errors.New("GetMultiUploadRecord get record for video is nil or size is wrong")
//		return
//	}
//	size = record.Size
//	return
//}
func GetVideoSnapShot(ctx context.Context, key string, du float64, code string, kind int, proj string) (snapId int64, err error) {
	idCh := make(chan int64, 1)
	go GetDistributedId(idCh)
	stsName := GetMtsStsName(kind) //使用具有mts权限的sts信息
	ossInfo, err := GetAliOssClient(ctx, stsName)
	if err != nil {
		return
	}
	snapId = <-idCh
	if snapId <= 0 {
		err = errors.New("GetVideoSnapShot.GetDistributedId get id is nil")
		return
	}
	err = GetOssCallbackSnapShot(ctx, ossInfo.Client, nil, key, ossInfo.Sts, snapId, du, code, ossInfo.Sts.Bucket, conf.C.Bucket.Resource, api.SnapConfTime)
	if err != nil {
		return
	}
	snapDoc, err := OrganizeSnapShotDoc(ctx, proj, snapId)
	if err != nil {
		return
	}
	err = DaoByQid.InsertResourceDoc(snapId, snapDoc)
	if err != nil {
		return
	}
	return
}
func GetUploadMp4DBDoc(ctx context.Context, key, qetag, proj string, kind, w, h int, size int64, du float64, code, fmt string) (qDoc *api.XngResourceInfoDoc, err error) {
	resId, err := strconv.ParseInt(key, 10, 64)
	if err != nil {
		return
	}
	upt := time.Now().UnixNano() / 1e6
	qDoc = &api.XngResourceInfoDoc{
		ResId: resId,
		Type:  kind,
		Size:  size,
		QeTag: qetag,
		Upt:   upt,
		Mt:    upt,
		Ct:    upt,
		Src:   proj,
		Ort:   1,
		W:     w,
		H:     h,
		Ref:   1,
		Du:    du,
	}
	if kind == api.ResourceTypeVideo { //上传完成的资源暂时默认已完成转码
		qDoc.TransCode = &ResStatusTrans
	} else if kind == api.ResourceTypeAlbum {
		qDoc.TransCode = &ResStatusMade
	}
	qDoc.Code = code
	qDoc.Fmt = fmt
	snapId, err := GetVideoSnapShot(ctx, key, du, code, kind, proj)
	if err != nil {
		return
	}
	qDoc.Cover = snapId
	qDoc.CoverTp = VideoCoverType
	return
}

type ResInfoWithCoverURLData struct {
	Status int                     `json:"status"` //status取值：为0时代表获取信息失败，为1时表示成功并附带资源信息
	Data   *api.ResDocWithCoverUrl `json:"data"`
}

func NotifyMediaInfoMq(ctx context.Context, prod int, proj, serviceName string, data *api.ResDocWithCoverUrl, status int) (err error) {
	topic, tag, err := utils.GetMediaInfoTopic(proj, prod, serviceName)
	if err != nil {
		return
	}
	mqData := ResInfoWithCoverURLData{
		Status: status,
		Data:   data,
	}
	err = callbackMQ.NotifyByMq(ctx, topic, tag, mqData)
	if err != nil {
		return
	}
	return
}
func NotifyVideoTransMq(ctx context.Context, serviceName string, data *api.ResDocWithCoverUrl, status int) (err error) {
	topic, tag := utils.GetVideoTransTopic(serviceName)
	mqData := ResInfoWithCoverURLData{
		Status: status,
		Data:   data,
	}
	err = callbackMQ.NotifyByMq(ctx, topic, tag, mqData)
	if err != nil {
		return
	}
	return
}
func HandleXngResourcesInfo(ctx context.Context, key, proj string, kind int, qetag, filePath string) (qdoc *api.XngResourceInfoDoc, err error) {
	stsName := GetMtsStsName(kind)
	mtsInfo, err := GetAliMtsClient(ctx, stsName)
	if err != nil {
		return
	}
	switch kind { //转码成功，根据转码的资源类型进行处理
	case api.ResourceTypeAlbum, api.ResourceTypeVideo:
		//获取视频信息
		doc, errIgnore := GetOssCallbackVideoDoc(ctx, mtsInfo.Client, mtsInfo.Sts, key, filePath, qetag, proj, mtsInfo.Sts.Bucket, kind)
		if errIgnore != nil {
			err = errIgnore
			return
		}
		//获取截图
		snapId, errIgnore := GetVideoSnapShot(ctx, key, doc.Du, doc.Code, kind, proj)
		if errIgnore != nil {
			err = errIgnore
			return
		}
		doc.Cover = snapId
		doc.CoverTp = VideoCoverType
		qdoc = doc
	case api.ResourceTypeGuideVideo:
		doc, errIgnore := GetOssCallbackVideoDoc(ctx, mtsInfo.Client, mtsInfo.Sts, key, filePath, qetag, proj, mtsInfo.Sts.Bucket, kind)
		if errIgnore != nil {
			err = errIgnore
			return
		}
		qdoc = doc
		return //导播视频不存储资源信息
	default:
		return
	}
	//数据存库
	resKey := strconv.FormatInt(qdoc.ResId, 10)
	err = DaoByQid.InsertResourceDoc(qdoc.ResId, qdoc)
	if err != nil {
		return
	}
	_ = resourceDao.SetCache(ctx, resKey, qdoc, api.CacheMiddleTime)
	return
}
func UpdateResTransRecord(ctx context.Context, oldID, transID, tplID string) (err error) {
	var qdoc *api.XngResourceInfoDoc
	qdoc, err = ByID(ctx, oldID)
	if err != nil {
		return
	}
	if qdoc == nil { //导播等类型资源不存储记录
		return
	}
	rateData := api.RateInfo{
		TplID: tplID,
		ID:    transID,
	}
	qdoc.VRate = append(qdoc.VRate, rateData)
	qry := bson.M{"_id": oldID}
	id, err := strconv.ParseInt(oldID, 10, 64)
	if err != nil {
		return
	}
	updata := bson.M{"$set": bson.M{"vrate": qdoc.VRate}}
	err = DaoByQid.UpdateResourceDoc(id, qry, updata)
	if err != nil {
		xlog.Error("update resource doc id:%v, error:%v", id, err)
		return
	}
	return
}
