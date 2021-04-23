package cron_task

import (
	"context"
	"encoding/xml"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	resourceDao "xgit.xiaoniangao.cn/xngo/service/resources_center/dao/resource"
	ServiceSts "xgit.xiaoniangao.cn/xngo/service/resources_center/service/sts"
)

func AbortMultiUpload(key, uploadID string, stsData *ServiceSts.UploadToken) (err error) {

	endpoint := stsData.Endpoint
	if conf.Env == lib.PROD {
		endpoint = stsData.EndpointInternal
	}
	client, err := oss.New(endpoint, stsData.AccessKey, stsData.SecretKey, oss.SecurityToken(stsData.SecurityToken))
	if err != nil {
		return
	}
	bucket, err := client.Bucket(stsData.Bucket)
	if err != nil {
		return
	}
	imur := oss.InitiateMultipartUploadResult{
		XMLName:  xml.Name{},
		Bucket:   stsData.Bucket,
		Key:      key,
		UploadID: uploadID,
	}
	err = bucket.AbortMultipartUpload(imur)
	if err != nil {
		return
	}
	return
}

//定时清理无效的上传分片
func RegularCleanMultiParts() {
	if conf.Env == lib.PROD {
		ok, err := resourceDao.SetRunningRoutine()
		if err != nil {
			xlog.Error("RegularCleanMultiParts.SetRunningRoutine failed, err:%v", err)
			return
		}
		if ok == false { //存在工作线程则退出
			xlog.Debug("RegularCleanMultiParts.SetRunningRoutine already running")
			return
		}
		xlog.Debug("RegularCleanMultiParts.SetRunningRoutine start")
		//dayNum := time.Now().Day()
		//ind := dayNum%api.CronMachineCnt + 1
		//hostName, err := os.Hostname()
		//if err != nil {
		//	xlog.Error("failed to get hostName, err:%v", err)
		//	return
		//}
		//if hostName != api.CleanTaskMap[ind] {
		//	return
		//}
	}
	score := time.Now().Add(-time.Hour * 24).Unix()
	records, err := resourceDao.GetRangMultiAbortRecord(score)
	if err != nil {
		xlog.Error("failed to get multi abort record, err:%v", err)
		return
	}
	if records == nil {
		xlog.Debug("multiUpload abort records is nil")
		return
	}
	stsData, err := ServiceSts.GetUpToken(context.Background(), api.StsForUserUpload)
	if err != nil {
		xlog.Error("failed to get upToken, err:%v", err)
		return
	}
	if stsData == nil {
		xlog.Error("failed to get upToken, upToken is nil")
		return
	}
	for _, record := range records {
		nerr := resourceDao.DelMultiUploadRecord(record.QeTag)
		if nerr != nil {
			xlog.Error("failed to del multi upload record, err:%v, DB_record:%v", nerr, record)
			errIngro := resourceDao.AddMultiAbortRecord(record) //删除失败则更新score，尝试下一次进行删除
			if errIngro != nil {
				xlog.Error("failed to add multi abort record, err:%v, DB_record:%v", nerr, record)
			}
			continue
		}
		nerr = AbortMultiUpload(record.Key, record.UploadID, stsData)
		if nerr != nil {
			xlog.Error("failed to abort multiUpload task, err:%v, DB_record:%v", nerr, record)
			continue
		}
	}
	err = resourceDao.DelRangeMultiAbortRecord(score)
	if err != nil {
		xlog.Error("failed to clean multiParts, err:%v", err)
		return
	}
	xlog.Debug("success clean multiParts, partsInfo:%v", records)
}
