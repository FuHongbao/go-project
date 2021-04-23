package resource

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByQid"
	resourceDao "xgit.xiaoniangao.cn/xngo/service/resources_center/dao/resource"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/resource"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/sts"
	urlService "xgit.xiaoniangao.cn/xngo/service/resources_center/service/url"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils"
)

type ReqGetUploadInfo struct {
	Kind      int    `json:"type"`
	QeTag     string `json:"qetag"`
	Product   int    `json:"prod"`       //产品 1 xng 2 xbd 3 tia
	Project   string `json:"proj"`       //ma app ...
	NoBack    bool   `json:"no_back"`    //是否取消资源存库，默认对资源进行信息存储，若指定为true时将不存储资源信息，并且不进行mq回调
	NoMq      bool   `json:"no_mq"`      //是否取消mq回调，为true时将不进行mq回调，默认mq有回调
	MusicName string `json:"music_name"` //音乐类资源的歌曲名字
}

type RespGetUploadInfo struct {
	Host             string                  `json:"host"`
	InternalHost     string                  `json:"internal_host"`
	UploadInfo       api.UploadInfo          `json:"upload_info"`
	UploadCustomInfo api.CallbackCustomParam `json:"upload_custom_info"`
	ID               string                  `json:"id"`
	ExpireSec        int                     `json:"expire_sec"`
}

// GetUploadInfo
// @Summary 资源中心获取上传配置接口
// @Description 业务端调用此接口获取上传配置，进行前端直传
// @Accept  json
// @Produce  json
// @Param body api.GetUploadConfReq true "请求相关的参数"
// @success 200 {object} api.UploadConfCommonResp{} "返回JSON数据"
// @Router /resource/get_upload_info [post]
func GetUploadInfo(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req ReqGetUploadInfo
	if !xc.GetReqObject(&req) {
		return
	}
	if checkResType(req.Kind) == false || req.QeTag == "" || req.Project == "" || req.Product <= 0 {
		xc.ReplyFail(lib.CodePara)
		return
	}
	uploadInfo, err := resource.GetUploadConfInfo(xc, req.Kind, req.NoBack)
	if err != nil {
		xlog.ErrorC(xc, "failed to GetUploadConfInfo, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	mqNoback := 0
	if req.NoMq == true {
		mqNoback = 1
	}
	resp := RespGetUploadInfo{
		Host:         uploadInfo.Host,
		InternalHost: uploadInfo.InternalHost,
		UploadInfo:   uploadInfo,
		UploadCustomInfo: api.CallbackCustomParam{
			QeTag:     req.QeTag,
			Kind:      req.Kind,
			Product:   req.Product,
			Project:   req.Project,
			NoMq:      mqNoback,
			MusicName: req.MusicName,
		},
		ID:        uploadInfo.Key,
		ExpireSec: uploadInfo.ExpireSec,
	}

	xc.ReplyOK(resp)
}

func GetStaticResUploadConfig(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req api.ReqStaticUploadConf
	if !xc.GetReqObject(&req) {
		return
	}
	stsData, err := sts.GetUpToken(xc, api.StsForStaticUpload)
	if err != nil {
		xlog.ErrorC(xc, "failed to GetUpToken, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if stsData == nil {
		xlog.ErrorC(xc, "failed to GetUpToken, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	ObjectKey := api.StaticUploadMap[req.Prod] + "/" + req.YWSide + "/" + req.Path + "/" + req.FileName
	resp := resource.GetStaticUploadConf(xc, ObjectKey, stsData)
	ossHeaders := "x-oss-security-token:" + stsData.SecurityToken + "\n"
	ObjectPath := "/" + stsData.Bucket + "/" + ObjectKey
	signature, err := resource.GetHeaderSignature(resp.Method, stsData.SecretKey, "", req.ContentType, resp.Date, ossHeaders, ObjectPath)
	if err != nil {
		xlog.ErrorC(xc, "failed to GetUploadConfInfo, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resp.Authorization = resource.GetAuthorization(stsData.AccessKey, signature)
	xc.ReplyOK(resp)
}

type ResDoc struct {
	ResId string  `json:"id"`
	Type  int     `json:"ty"`
	Size  int64   `json:"size"`
	QeTag string  `json:"qetag"`
	Upt   int64   `json:"upt"`
	Fmt   string  `json:"fmt"`
	W     int     `json:"w"`
	H     int     `json:"h"`
	Du    float64 `json:"du,omitempty"`
	Cover string  `json:"cover,omitempty"`
	Code  string  `json:"code,omitempty"`
	Ort   int     `json:"ort,omitempty"`
}

func GetMultiUploadConfig(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req api.ReqGetMultiUploadConf
	if !xc.GetReqObject(&req) {
		return
	}
	if len(req.QeTag) <= 8 || req.Kind <= 0 || req.QeTag == "" || req.Size <= 0 {
		xc.ReplyFail(lib.CodePara)
		return
	}
	req.QeTag = utils.GetQetagByResType(req.Kind, req.QeTag)
	status, doc, err := resource.GetUploadStatus(xc, req.QeTag)
	if err != nil {
		xlog.ErrorC(xc, "failed to GetUploadStatus, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if status == resource.UploadStatusSuccess && doc.Type == req.Kind { //若资源存在且类型标识一致则直接返回资源信息 (解决旧资源类型的干扰)
		ty := doc.Type
		if ty == api.ResourceTypeGroupImg {
			ty = api.ResourceTypeImg
		}
		respDoc := ResDoc{
			ResId: fmt.Sprintf("%d", doc.ResId),
			Type:  ty,
			Size:  doc.Size,
			QeTag: doc.QeTag,
			Upt:   doc.Upt,
			Fmt:   doc.Fmt,
			W:     doc.W,
			H:     doc.H,
			Du:    doc.Du,
			Cover: fmt.Sprintf("%d", doc.Cover),
			Code:  doc.Code,
			Ort:   doc.Ort,
		}
		respData := &xng.XResp{
			Ret:  lib.CodeExist,
			Data: respDoc,
		}
		xc.Reply(http.StatusOK, respData)
		return
	}
	stsName := resource.GetUploadStsName(req.Kind)
	/*
		stsData, err := sts.GetUpToken(xc, stsName)
		if err != nil {
			xlog.ErrorC(xc, "failed to GetUpToken, req:%v, err:%v", req, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		if stsData == nil {
			xlog.ErrorC(xc, "failed to GetUpToken, req:%v, err:%v", req, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
	*/
	ossInfo, err := resource.GetAliOssClient(xc, stsName)
	if err != nil {
		xlog.ErrorC(xc, "GetMultiUploadConfig.GetAliOssClient, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	var resp api.RespMultiUploadConf
	if status == resource.UploadStatusInterrupt { //处于上传中断状态，已存在上传事件且可能有部分上传成功分片（此流程不能成功则立即清除相关记录，走新上传流程）
		record, errIgnore := resourceDao.GetMultiUploadRecord(req.QeTag)
		if errIgnore != nil {
			xlog.ErrorC(xc, "Get MultiUpload Record err, err:%v", errIgnore)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		if record == nil {
			xlog.ErrorC(xc, "MultiUpload Record is nil, req:%v", req)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		ok := true
		parts, chunks, chunkCnt, errIgnore := resource.GetResumeConfigs(xc, record.Key, record.UploadID, req.Size, ossInfo.Client, ossInfo.Sts.Bucket) //存在阿里云UploadID失效，但本地redis记录还存在的问题，此处查询失败删除记录走新上传逻辑
		if errIgnore != nil {
			xlog.ErrorC(xc, "Get Resume Configs err, req:%v, err:%v", req, errIgnore)
			errIgnore = resource.DelMultiRecord(req.QeTag, record.Key, record.UploadID)
			if errIgnore != nil {
				xlog.ErrorC(xc, "failed to DelMultiRecord, req:%v, err:%v", req, errIgnore)
				xc.ReplyFail(lib.CodeSrv)
				return
			}
			ok = false
		}
		if ok {
			resp.Key = record.Key
			resp.UploadID = record.UploadID
			resp.Chunks = chunks
			resp.ChunkCnt = chunkCnt
			resp.Parts = parts
			xc.ReplyOK(resp)
			return
		}
	}
	//状态为未上传,生成新的上传事件
	ch := make(chan int64, 1)
	go resource.GetDistributedId(ch)
	chunks, chunkCnt := resource.GetNewChunkInfo(req.Size)
	resp.Chunks = chunks
	resp.ChunkCnt = chunkCnt
	qid := <-ch
	if qid <= 0 {
		xlog.ErrorC(xc, "failed to GetDistributedId, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resp.Key = strconv.FormatInt(qid, 10)
	if resp.Key == "" {
		xlog.ErrorC(xc, "failed to FormatInt, qid:%v, err:%v", qid, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	startTime := time.Now()
	uploadID, err := resource.InitMultiUpload(xc, resp.Key, ossInfo.Client, ossInfo.Sts.Bucket)
	if err != nil {
		xlog.ErrorC(xc, "initMultiUpload err, use time:%v, err:%v", time.Since(startTime), err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xlog.DebugC(xc, "initMultiUpload use time : %v", time.Since(startTime))
	resp.UploadID = uploadID
	err = resource.AddMultiRecord(req.QeTag, uploadID, resp.Key, req.Size)
	if err != nil {
		xlog.ErrorC(xc, "AddMultiRecord err, use time:%v, err:%v", time.Since(startTime), err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.ReplyOK(resp)
}

func GetDefaultContentType(kind int) string {
	switch kind {
	case api.ResourceTypeImg, api.ResourceTypeGroupImg:
		return api.ContentTypeImg
	case api.ResourceTypeVideo, api.ResourceTypeAlbum:
		return api.ContentTypeVideo
	case api.ResourceTypeMusic, api.ResourceTypeVoice:
		return api.ContentTypeMusic
	case api.ResourceTypeTxt, api.ResourceTypeLyric:
		return api.ContentTypeTxt
	}
	return ""
}

func CheckMultiUploadResult(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req api.ReqCheckMultiUpload
	if !xc.GetReqObject(&req) {
		return
	}
	if req.QeTag == "" || req.Key == "" || req.Project == "" || req.Product <= 0 || req.UploadID == "" {
		xc.ReplyFail(lib.CodePara)
		return
	}
	if req.Kind != api.ResourceTypeImg && req.Kind != api.ResourceTypeMusic && req.Kind != api.ResourceTypeVoice && req.Kind != api.ResourceTypeVideo && req.Kind != api.ResourceTypeLyric && req.Kind != api.ResourceTypeTxt && req.Kind != api.ResourceTypeAlbum {
		xc.ReplyFail(lib.CodePara)
		return
	}
	req.QeTag = utils.GetQetagByResType(req.Kind, req.QeTag)
	var resp api.RespCheckMultiUpload
	status, doc, err := resource.GetUploadStatus(xc, req.QeTag)
	if err != nil {
		xlog.ErrorC(xc, "CheckMultiUploadResult.failed to GetUploadStatus, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if status == resource.UploadStatusSuccess && doc.Type == req.Kind { //若资源存在则直接返回资源信息
		ty := doc.Type
		if ty == api.ResourceTypeGroupImg {
			ty = api.ResourceTypeImg
		}
		resp.Status = 1
		resp.Info = api.ResourceInfo{
			ResId: fmt.Sprintf("%d", doc.ResId),
			Type:  doc.Type,
			Size:  doc.Size,
			QeTag: doc.QeTag,
			Upt:   doc.Upt,
			Fmt:   doc.Fmt,
			W:     doc.W,
			H:     doc.H,
			Du:    doc.Du,
			Cover: fmt.Sprintf("%d", doc.Cover),
			Code:  doc.Code,
			Ort:   doc.Ort,
		}
		xc.ReplyOK(resp)
		return
	}
	if req.ContentType == "" {
		req.ContentType = GetDefaultContentType(req.Kind)
	}

	stsName := resource.GetMtsStsName(req.Kind)
	/*
		stsData, err := sts.GetUpToken(xc, stsName)
		if err != nil {
			xlog.ErrorC(xc, "failed to GetUpToken, req:%v, err:%v", req, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		if stsData == nil {
			xlog.ErrorC(xc, "failed to GetUpToken, sts data is nil")
			xc.ReplyFail(lib.CodeSrv)
			return
		}
	*/
	ossInfo, err := resource.GetAliOssClient(xc, stsName)
	if err != nil {
		xlog.ErrorC(xc, "CheckMultiUploadResult.GetAliOssClient failed, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	mtsInfo, err := resource.GetAliMtsClient(xc, stsName)
	if err != nil {
		xlog.ErrorC(xc, "CheckMultiUploadResult.GetAliMtsClient failed, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if req.Parts == nil {
		req.Parts, err = resource.GetEtagList(xc, req.Key, req.UploadID, ossInfo.Sts, ossInfo.Client)
		if err != nil {
			xlog.ErrorC(xc, "failed to GetEtagList, req:%v, err:%v", req, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
	}
	ret, err := resource.MergeMultiParts(xc, &req, ossInfo.Sts, ossInfo.Client)
	if err != nil {
		xlog.ErrorC(xc, "failed to MergeMultiParts, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = resource.DelMultiRecord(req.QeTag, req.Key, req.UploadID)
	if err != nil {
		xlog.ErrorC(xc, "failed to DelMultiRecord, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if ret == false {
		resp.Status = 0
		xc.ReplyOK(resp)
		return
	}
	//设置资源的contentType
	//errIgnore := resource.SetContentTypeForCommon(xc, req.Key, stsData, req.ContentType)
	errIgnore := resource.SetOssCallBackContentType(xc, ossInfo.Client, req.Key, ossInfo.Sts.Bucket, req.ContentType)
	if errIgnore != nil {
		xlog.ErrorC(xc, "set resource content-type error:%v, fileName:%v", errIgnore, req.Key)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resDoc, ok, err := resource.GetMultiUplaodResInfo(xc, req, ossInfo.Sts, ossInfo.Client, mtsInfo.Client)
	if err != nil {
		xlog.ErrorC(xc, "failed to GetMultiUplaodResInfo, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if !ok {
		resp.Status = 0
		xc.ReplyOK(resp)
		return
	}
	err = resource.AddNewMediaInfoToDB(resDoc)
	if err != nil {
		xlog.ErrorC(xc, "failed to AddNewMediaInfoToDB, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = resource.AddMediaInfoCache(xc, resDoc, req.Key)
	if err != nil {
		xlog.ErrorC(xc, "failed to AddMediaInfoCache, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resID := strconv.FormatInt(resDoc.ResId, 10)
	coverID := strconv.FormatInt(resDoc.Cover, 10)
	resp.Status = 1
	resp.Info = api.ResourceInfo{
		ResId: resID,
		Type:  resDoc.Type,
		Size:  resDoc.Size,
		QeTag: resDoc.QeTag,
		Upt:   resDoc.Upt,
		Fmt:   resDoc.Fmt,
		W:     resDoc.W,
		H:     resDoc.H,
		Du:    resDoc.Du,
		Cover: coverID,
		Code:  resDoc.Code,
		Ort:   resDoc.Ort,
	}
	xc.ReplyOK(resp)
}

func checkResType(resType int) bool {
	switch resType {
	case api.ResourceTypeImg, api.ResourceTypeMusic, api.ResourceTypeVoice, api.ResourceTypeVideo, api.ResourceTypeLyric, api.ResourceTypeTxt, api.ResourceTypeAlbum, api.ResourceTypeLive, api.ResourceTypeGuideVideo, api.ResourceTypeAPPLog:
		return true
	default:
		return false
	}
}
func ValidMediaInfo(info string) (mediaInfo api.MultiMediaInfo, err error) {
	err = json.Unmarshal([]byte(info), &mediaInfo)
	return
}
func CheckMultiUploadResultV2Param(req *api.ReqCheckMultiUpload) (ret bool, mediaInfo api.MultiMediaInfo, msg string) {
	if req.QeTag == "" {
		msg = "CheckMultiUploadResultV2.Qetag字段校验不通过"
		return
	}
	if req.Key == "" {
		msg = "CheckMultiUploadResultV2.Key字段校验不通过"
		return
	}
	if req.Project == "" {
		msg = "CheckMultiUploadResultV2.Proj字段校验不通过"
		return
	}
	if req.Product <= 0 {
		msg = "CheckMultiUploadResultV2.Prod字段校验不通过"
		return
	}
	if req.UploadID == "" {
		msg = "CheckMultiUploadResultV2.UploadID字段校验不通过"
		return
	}
	if checkResType(req.Kind) == false {
		msg = "CheckMultiUploadResultV2.Type资源类型验证不通过"
		return
	}
	if req.Kind != api.ResourceTypeVideo && req.Kind != api.ResourceTypeAlbum {
		ret = true
		return
	}
	if req.MediaInfo == "" {
		msg = "CheckMultiUploadResultV2.Media_info字段校验不通过"
		return
	}
	_ = json.Unmarshal([]byte(req.MediaInfo), &mediaInfo)
	if mediaInfo.W <= 0 || mediaInfo.H <= 0 || mediaInfo.Du <= 0 {
		msg = "CheckMultiUploadResultV2.Media_info解析验证参数不通过"
		return
	}
	ret = true
	return
}
func CheckMultiUploadResultV2(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req api.ReqCheckMultiUpload
	if !xc.GetReqObject(&req) {
		return
	}
	ok, mediaInfo, msg := CheckMultiUploadResultV2Param(&req)
	if ok == false {
		respData := &xng.XResp{
			Ret: lib.CodePara,
			Msg: msg,
		}
		xc.Reply(http.StatusOK, respData)
		return
	}
	var err error
	req.QeTag = utils.GetQetagByResType(req.Kind, req.QeTag)
	//存在重复调用该接口的情况，先验证资源上传状态
	var resp api.RespCheckMultiUpload
	status, doc, err := resource.GetUploadStatus(xc, req.QeTag)
	if err != nil {
		xlog.ErrorC(xc, "CheckMultiUploadResultV2.GetUploadStatus failed, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if status == resource.UploadStatusSuccess { //若资源存在则直接返回资源信息
		ty := doc.Type
		if ty == api.ResourceTypeGroupImg {
			ty = api.ResourceTypeImg
		}
		resp.Status = 1
		resp.Info = api.ResourceInfo{
			ResId: fmt.Sprintf("%d", doc.ResId),
			Type:  doc.Type,
			Size:  doc.Size,
			QeTag: doc.QeTag,
			Upt:   doc.Upt,
			Fmt:   doc.Fmt,
			W:     doc.W,
			H:     doc.H,
			Du:    doc.Du,
			Cover: fmt.Sprintf("%d", doc.Cover),
			Code:  doc.Code,
			Ort:   doc.Ort,
		}
		xc.ReplyOK(resp)
		return
	}
	//为新上传的资源时，验证一下上传记录，上传记录不存在的为已合并或合并失败，都不继续进行
	record, err := resourceDao.GetMultiUploadRecord(req.QeTag)
	if err != nil {
		xlog.ErrorC(xc, "CheckMultiUploadResultV2.GetMultiUploadRecord err, req:[%v], err:[%v]", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if record == nil {
		resp.Status = 0
		xc.ReplyOK(resp)
		return
	}
	if mediaInfo.Size == 0 { //未传入size字段的，从记录中提取
		//mediaInfo.Size, err = resource.GetMultiRecordSize(xc, req.QeTag, req.Kind)
		//if err != nil {
		//	xlog.ErrorC(xc, "CheckMultiUploadResultV2.GetMultiRecordSize err, req:[%v], err:[%v]", req, err)
		//	xc.ReplyFail(lib.CodeSrv)
		//	return
		//}
		mediaInfo.Size = record.Size
	}
	if req.ContentType == "" {
		req.ContentType = resource.GetResContentType(req.Kind)
	}
	stsName := resource.GetMtsStsName(req.Kind)
	ossInfo, err := resource.GetAliOssClient(xc, stsName)
	if err != nil {
		xlog.ErrorC(xc, "CheckMultiUploadResultV2.GetAliOssClient failed, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	mtsInfo, err := resource.GetAliMtsClient(xc, stsName)
	if err != nil {
		xlog.ErrorC(xc, "CheckMultiUploadResultV2.GetAliMtsClient failed, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if req.Parts == nil {
		req.Parts, err = resource.GetEtagList(xc, req.Key, req.UploadID, ossInfo.Sts, ossInfo.Client)
		if err != nil {
			xlog.ErrorC(xc, "failed to GetEtagList, req:%v, err:%v", req, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
	}
	ret, err := resource.MergeMultiParts(xc, &req, ossInfo.Sts, ossInfo.Client)
	if err != nil {
		xlog.ErrorC(xc, "failed to MergeMultiParts, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = resource.DelMultiRecord(req.QeTag, req.Key, req.UploadID) //合并后upload_id不再存在，删除记录
	if err != nil {
		xlog.ErrorC(xc, "failed to DelMultiRecord, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if ret == false {
		resp.Status = 0
		xc.ReplyOK(resp)
		return
	}
	//设置资源的contentType
	errIgnore := resource.SetOssCallBackContentType(xc, ossInfo.Client, req.Key, ossInfo.Sts.Bucket, req.ContentType)
	if errIgnore != nil {
		xlog.ErrorC(xc, "set resource content-type error:%v, fileName:%v", errIgnore, req.Key)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resDoc, err := resource.GetMultiUplaodResInfoV2(xc, req, ossInfo.Sts, ossInfo.Client, mtsInfo.Client, &mediaInfo)
	if err != nil {
		xlog.ErrorC(xc, "failed to GetMultiUplaodResInfo, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = resource.AddNewMediaInfoToDB(resDoc)
	if err != nil {
		xlog.ErrorC(xc, "failed to AddNewMediaInfoToDB, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = resource.AddMediaInfoCache(xc, resDoc, req.Key)
	if err != nil {
		xlog.ErrorC(xc, "failed to AddMediaInfoCache, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resID := strconv.FormatInt(resDoc.ResId, 10)
	coverID := strconv.FormatInt(resDoc.Cover, 10)
	resp.Status = 1
	resp.Info = api.ResourceInfo{
		ResId: resID,
		Type:  resDoc.Type,
		Size:  resDoc.Size,
		QeTag: resDoc.QeTag,
		Upt:   resDoc.Upt,
		Fmt:   resDoc.Fmt,
		W:     resDoc.W,
		H:     resDoc.H,
		Du:    resDoc.Du,
		Cover: coverID,
		Code:  resDoc.Code,
		Ort:   resDoc.Ort,
	}
	xc.ReplyOK(resp)
}

type ReqOpUploadInfo struct {
	QeTag string `json:"qetag"`
	Kind  int    `json:"type"`
}
type RespOpUploadInfo struct {
	Endpoint      string `json:"endpoint"`
	EndpointInter string `json:"endpoint_internal"`
	Bucket        string `json:"bucket"`
	AccessKey     string `json:"accessKey"`
	SecretKey     string `json:"secretKey"`
	SecurityToken string `json:"securityToken"`
	Region        string `json:"region"`
	Qid           string `json:"id"`
}

func GetOpUploadInfo(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req ReqOpUploadInfo
	if !xc.GetReqObject(&req) {
		return
	}
	if req.QeTag == "" || checkResType(req.Kind) == false {
		xc.ReplyFail(lib.CodePara)
		return
	}
	req.QeTag = utils.GetQetagByResType(req.Kind, req.QeTag)
	qid, err := resource.ByQeTag(xc, req.QeTag)
	if err != nil {
		xlog.ErrorC(xc, "failed to get qid by qetag, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if qid != "" {
		doc, err := resource.ByID(xc, qid)
		if err != nil {
			xlog.ErrorC(xc, "failed to get media info by qid, req:%v, err:%v", req, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		if doc != nil {
			cover := fmt.Sprintf("%d", doc.Cover)
			url, urlInternal := urlService.GetImageURL(xc, cover, "imageMogr2/")
			respDoc := api.ResDocWithCoverUrl{
				ResId:            fmt.Sprintf("%d", doc.ResId),
				Type:             doc.Type,
				Size:             doc.Size,
				QeTag:            doc.QeTag,
				Upt:              doc.Upt,
				Fmt:              doc.Fmt,
				W:                doc.W,
				H:                doc.H,
				Du:               doc.Du,
				Cover:            cover,
				Code:             doc.Code,
				Ort:              doc.Ort,
				CoverUrl:         url,
				CoverUrlInternal: urlInternal,
			}
			respData := &xng.XResp{
				Ret:  lib.CodeExist,
				Data: respDoc,
			}
			xc.Reply(http.StatusOK, respData)
			return
		}
	}
	ch := make(chan int64, 1)
	go resource.GetDistributedId(ch)
	stsName := resource.GetUploadStsName(req.Kind)
	ossInfo, err := resource.GetAliOssClient(xc, stsName)
	if err != nil {
		xlog.ErrorC(xc, "CheckMultiUploadResultV2.GetAliOssClient failed, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xlog.DebugC(xc, "conf:[%v]", conf.C.Sts[stsName])
	id := <-ch
	ind := strings.Index(ossInfo.Sts.Endpoint, ".")
	region := ossInfo.Sts.Endpoint[0:ind]
	resp := RespOpUploadInfo{
		Endpoint:      ossInfo.Sts.Endpoint,
		EndpointInter: ossInfo.Sts.EndpointInternal,
		Bucket:        ossInfo.Sts.Bucket,
		AccessKey:     ossInfo.Sts.AccessKey,
		SecretKey:     ossInfo.Sts.SecretKey,
		SecurityToken: ossInfo.Sts.SecurityToken,
		Region:        region,
		Qid:           fmt.Sprintf("%d", id),
	}
	xc.ReplyOK(resp)
}

type ReqCheckOpUpload struct {
	Key   string  `json:"id"`
	QeTag string  `json:"qetag"`
	Type  int     `json:"type"`
	Prod  int     `json:"prod"`
	Proj  string  `json:"proj"`
	W     int     `json:"w"`
	H     int     `json:"h"`
	Size  int64   `json:"size"`
	Du    float64 `json:"du"`
	Code  string  `json:"code"`
	Fmt   string  `json:"fmt"`
}
type RespCheckOpUpload api.ResDocWithCoverUrl

func CheckOpUploadResult(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req ReqCheckOpUpload
	if !xc.GetReqObject(&req) {
		return
	}
	if req.Key == "" || req.QeTag == "" || req.Code == "" || req.Du <= 0 || req.H <= 0 || req.W <= 0 || req.Fmt == "" || req.Proj == "" || req.Prod <= 0 || checkResType(req.Type) == false {
		xc.ReplyFail(lib.CodePara)
		return
	}
	req.QeTag = utils.GetQetagByResType(req.Type, req.QeTag)
	doc, err := resource.ByID(xc, req.Key)
	if err != nil {
		xlog.ErrorC(xc, "failed to get media info by qid, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if doc != nil {
		cover := fmt.Sprintf("%d", doc.Cover)
		url, urlInternal := urlService.GetImageURL(xc, cover, "imageMogr2/thumbnail/750x500/format/jpg")
		respDoc := api.ResDocWithCoverUrl{
			ResId:            fmt.Sprintf("%d", doc.ResId),
			Type:             doc.Type,
			Size:             doc.Size,
			QeTag:            doc.QeTag,
			Upt:              doc.Upt,
			Fmt:              doc.Fmt,
			W:                doc.W,
			H:                doc.H,
			Du:               doc.Du,
			Cover:            cover,
			Code:             doc.Code,
			Ort:              doc.Ort,
			CoverUrl:         url,
			CoverUrlInternal: urlInternal,
		}
		xc.ReplyOK(respDoc)
		return
	}
	mType := resource.GetResContentType(req.Type)
	stsName := resource.GetMtsStsName(req.Type) //使用具有mts权限的sts信息
	ossInfo, err := resource.GetAliOssClient(xc, stsName)
	if err != nil {
		xlog.ErrorC(xc, "CheckOpUploadResult.GetAliOssClient failed, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = resource.SetOssCallBackContentType(xc, ossInfo.Client, req.Key, ossInfo.Sts.Bucket, mType)
	if err != nil {
		xlog.ErrorC(xc, "CheckOpUploadResult.SetOssCallBackContentType failed, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	switch req.Type {
	case api.ResourceTypeAlbum, api.ResourceTypeVideo:
		idCh := make(chan int64, 1)
		go resource.GetDistributedId(idCh)
		mtsInfo, errIgnore := resource.GetAliMtsClient(xc, stsName)
		if errIgnore != nil {
			xlog.ErrorC(xc, "CheckOpUploadResult.GetAliMtsClient failed, req:%v, err:%v", req, errIgnore)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		qDoc, errIgnore := resource.GetMultiBaseDoc(xc, req.Key, req.QeTag, req.Proj, req.Type, req.W, req.H, req.Du, req.Size)
		if errIgnore != nil {
			xlog.ErrorC(xc, "CheckOpUploadResult.GetMultiBaseDoc failed, req:%v, err:%v", req, errIgnore)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		qDoc.Code = req.Code
		qDoc.Fmt = req.Fmt
		snapId := <-idCh
		if snapId <= 0 {
			xlog.ErrorC(xc, "CheckOpUploadResult.GetDistributedId failed, req:%v, id is nil", req)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		errIgnore = resource.GetOssCallbackSnapShot(xc, ossInfo.Client, mtsInfo.Client, req.Key, mtsInfo.Sts, snapId, req.Du, req.Code, ossInfo.Sts.Bucket, conf.C.Bucket.Resource, api.SnapConfTime)
		if errIgnore != nil {
			xlog.ErrorC(xc, "CheckOpUploadResult.GetOssCallbackSnapShot failed, req:%v, err:%v", req, errIgnore)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		//获取截图信息并整理doc
		snapDoc, errIgnore := resource.OrganizeSnapShotDoc(xc, req.Proj, snapId)
		if errIgnore != nil {
			xlog.ErrorC(xc, "CheckOpUploadResult.OrganizeSnapShotDoc failed, req:%v, err:%v", req, errIgnore)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		errIgnore = DaoByQid.InsertResourceDoc(snapId, snapDoc)
		if errIgnore != nil {
			xlog.ErrorC(xc, "CheckOpUploadResult.InsertResourceDoc failed, req:%v, err:%v", req, errIgnore)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		qDoc.Cover = snapId
		qDoc.CoverTp = resource.VideoCoverType
		doc = qDoc
	}
	err = resource.AddNewMediaInfoToDB(doc)
	if err != nil {
		xlog.ErrorC(xc, "failed to AddNewMediaInfoToDB, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = resource.AddMediaInfoCache(xc, doc, req.Key)
	if err != nil {
		xlog.ErrorC(xc, "failed to AddMediaInfoCache, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	cover := fmt.Sprintf("%d", doc.Cover)
	url, urlInternal := urlService.GetImageURL(xc, cover, "imageMogr2/thumbnail/750x500/format/jpg")
	respDoc := api.ResDocWithCoverUrl{
		ResId:            fmt.Sprintf("%d", doc.ResId),
		Type:             doc.Type,
		Size:             doc.Size,
		QeTag:            doc.QeTag,
		Upt:              doc.Upt,
		Fmt:              doc.Fmt,
		W:                doc.W,
		H:                doc.H,
		Du:               doc.Du,
		Cover:            cover,
		Code:             doc.Code,
		Ort:              doc.Ort,
		CoverUrl:         url,
		CoverUrlInternal: urlInternal,
	}
	xc.ReplyOK(respDoc)
	return
}

type ReqCheckOpUploadResultV2 struct {
	ReqCheckOpUpload
	ProjectName string `json:"p_name"`
	ContentType string `json:"content_type"`
	UserData    string `json:"user_data"`
}
type RespCheckOpUploadResultV2 struct {
	Status int `json:"status"`
}

func CheckOpUploadResultV2(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req ReqCheckOpUploadResultV2
	if !xc.GetReqObject(&req) {
		return
	}
	if req.Key == "" || req.QeTag == "" || req.Proj == "" || req.Prod <= 0 || checkResType(req.Type) == false {
		xc.ReplyFail(lib.CodePara)
		return
	}
	req.QeTag = utils.GetQetagByResType(req.Type, req.QeTag)
	doc, err := resource.ByID(xc, req.Key)
	if err != nil {
		xlog.ErrorC(xc, "failed to get media info by qid, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	var resp RespCheckOpUploadResultV2
	if doc != nil {
		xlog.DebugC(xc, "ByID doc already exists, key:[%v]", doc.ResId)
		resDoc := api.ResDocWithCoverUrl{
			ResId:    fmt.Sprintf("%d", doc.ResId),
			Type:     doc.Type,
			Size:     doc.Size,
			QeTag:    doc.QeTag,
			Upt:      doc.Upt,
			Fmt:      doc.Fmt,
			W:        doc.W,
			H:        doc.H,
			Du:       doc.Du,
			Code:     doc.Code,
			Ort:      doc.Ort,
			UserData: req.UserData,
		}
		if doc.Cover > 0 {
			cover := fmt.Sprintf("%d", doc.Cover)
			url, urlInternal := urlService.GetImageURL(xc, cover, "imageMogr2/thumbnail/750x500/format/jpg")
			resDoc.Cover = cover
			resDoc.CoverUrl = url
			resDoc.CoverUrlInternal = urlInternal
		}
		err = resource.NotifyMediaInfoMq(xc, req.Prod, req.Proj, req.ProjectName, &resDoc, 1)
		if err != nil {
			xlog.ErrorC(xc, "CheckOpUploadResultV2.NotifyMediaInfoMq failed, req:%v, err:%v", req, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}
		resp.Status = 1
		xc.ReplyOK(resp)
		return
	}
	stsName := resource.GetMtsStsName(req.Type) //使用具有mts权限的sts信息
	ossInfo, err := resource.GetAliOssClient(xc, stsName)
	if err != nil {
		xlog.ErrorC(xc, "CheckOpUploadResultV2.GetAliOssClient failed, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = resource.SetOssCallBackContentType(xc, ossInfo.Client, req.Key, ossInfo.Sts.Bucket, req.ContentType)
	if err != nil {
		xlog.ErrorC(xc, "CheckOpUploadResultV2.SetOssCallBackContentType failed, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xlog.DebugC(xc, "SetOssCallBackContentType success")
	switch req.Type {
	case api.ResourceTypeAlbum, api.ResourceTypeVideo:
		if checkUploadMp4Info(&req) {
			xlog.DebugC(xc, "checkUploadMp4Info is true")
			qDoc, err := resource.GetUploadMp4DBDoc(xc, req.Key, req.QeTag, req.Proj, req.Type, req.W, req.H, req.Size, req.Du, req.Code, req.Fmt)
			if err != nil {
				xlog.ErrorC(xc, "CheckOpUploadResultV2.GetUploadMp4DBDoc failed, req:%v, err:%v", req, err)
				xc.ReplyFail(lib.CodeSrv)
				return
			}
			doc = qDoc
		} else { //非mp4资源或信息缺失的统一走阿里云接口获取资源信息
			xlog.DebugC(xc, "checkUploadMp4Info is false")
			mtsInfo, err := resource.GetAliMtsClient(xc, stsName)
			if err != nil {
				xlog.ErrorC(xc, "CheckOpUploadResultV2.GetAliMtsClient failed, req:%v, err:%v", req, err)
				xc.ReplyFail(lib.CodeSrv)
				return
			}
			endPoint := mtsInfo.Sts.Endpoint
			loc := endPoint[:len(endPoint)-13]
			data := api.MultiVideoUserData{
				UserData:    req.UserData,
				Qetag:       req.QeTag,
				UserService: req.ProjectName,
			}
			userByte, nerr := json.Marshal(data)
			if nerr != nil {
				xlog.ErrorC(xc, "CheckOpUploadResultV2.Marshal failed, req:%v, err:%v", req, err)
				xc.ReplyFail(lib.CodeSrv)
				return
			}
			nerr = resource.GetMediaInfoAsync(xc, mtsInfo.Client, mtsInfo.Sts.Bucket, loc, req.Key, "", req.Type, req.Prod, req.Proj, string(userByte), api.MTSJobTypeOpVideoInfo) //异步提交作业获取全部信息
			if nerr != nil {
				xlog.ErrorC(xc, "CheckOpUploadResultV2.GetMediaInfoAsync failed, req:%v, err:%v", req, err)
				xc.ReplyFail(lib.CodeSrv)
				return
			}
			//非mp4异步调用阿里云接口获取信息再进行存储
			resp.Status = 1
			xc.ReplyOK(resp)
			return
		}
	case api.ResourceTypeImg:
		qDoc, nerr := resource.GetImgInfo(xc, req.Key, api.ResourceTypeImg)
		if nerr != nil {
			err = nerr
			return
		}
		qDoc.QeTag = req.QeTag
		qDoc.Src = req.Proj
		doc = qDoc
	default:
		resp.Status = 0
		xc.ReplyOK(resp)
		return
	}
	//资源存库
	err = resource.AddNewMediaInfoToDB(doc)
	if err != nil {
		xlog.ErrorC(xc, "CheckOpUploadResultV2.failed to AddNewMediaInfoToDB, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = resource.AddMediaInfoCache(xc, doc, req.Key)
	if err != nil {
		xlog.ErrorC(xc, "CheckOpUploadResultV2.failed to AddMediaInfoCache, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resDoc := api.ResDocWithCoverUrl{
		ResId:    fmt.Sprintf("%d", doc.ResId),
		Type:     doc.Type,
		Size:     doc.Size,
		QeTag:    doc.QeTag,
		Upt:      doc.Upt,
		Fmt:      doc.Fmt,
		W:        doc.W,
		H:        doc.H,
		Du:       doc.Du,
		Code:     doc.Code,
		Ort:      doc.Ort,
		UserData: req.UserData,
	}
	if doc.Cover > 0 {
		cover := fmt.Sprintf("%d", doc.Cover)
		url, urlInternal := urlService.GetImageURL(xc, cover, "imageMogr2/thumbnail/750x500/format/jpg")
		resDoc.Cover = cover
		resDoc.CoverUrl = url
		resDoc.CoverUrlInternal = urlInternal
	}
	err = resource.NotifyMediaInfoMq(xc, req.Prod, req.Proj, req.ProjectName, &resDoc, 1)
	if err != nil {
		xlog.ErrorC(xc, "CheckOpUploadResultV2.NotifyMediaInfoMq failed, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resp.Status = 1
	xc.ReplyOK(resp)
	return
}

func checkUploadMp4Info(req *ReqCheckOpUploadResultV2) bool {
	if req.H > 0 && req.W > 0 && req.Du > 0 && req.Size > 0 && req.Code != "" && req.Fmt != "" {
		return true
	}
	return false
}
