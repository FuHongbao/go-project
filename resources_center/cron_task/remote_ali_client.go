package cron_task

import (
	"context"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/mts"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/sts"
)

func getOssClientRemote() {
	for k, _ := range conf.C.Sts {
		stsData, err := sts.GetUpToken(context.Background(), k)
		if err != nil {
			xlog.Error("getOssClientRemote.GetUpToken err:%v, key:%s", err, k)
			continue
		}
		if stsData == nil {
			xlog.Error("getOssClientRemote get sts information err, sts data is nil.")
			continue
		}
		endPoint := stsData.EndpointInternal
		if conf.Env != lib.PROD {
			endPoint = stsData.Endpoint
		}
		ossClient, err := oss.New(endPoint, stsData.AccessKey, stsData.SecretKey, oss.SecurityToken(stsData.SecurityToken))
		if err != nil {
			xlog.Error("getOssClientRemote.oss.New err:%v, key:%s", err, k)
			continue
		}
		api.StsOssClient[k] = &api.AliOssClient{
			Client: ossClient,
			Sts:    stsData,
		}
	}
}
func getMtsClientRemote() {
	for k, _ := range conf.C.Sts {
		stsData, err := sts.GetUpToken(context.Background(), k)
		if err != nil {
			xlog.Error("getMtsClientRemote.GetUpToken err:%v, key:%s", err, k)
			continue
		}
		if stsData == nil {
			xlog.Error("getMtsClientRemote get sts information err, sts data is nil.")
			continue
		}
		endPoint := stsData.EndpointInternal
		if conf.Env != lib.PROD {
			endPoint = stsData.Endpoint
		}
		regionID := endPoint[4 : len(endPoint)-13]
		mtsClient, err := mts.NewClientWithStsToken(regionID, stsData.AccessKey, stsData.SecretKey, stsData.SecurityToken)
		if err != nil {
			xlog.Error("getMtsClientRemote.NewClientWithStsToken error:%v, key:%s", err, k)
			continue
		}
		if conf.Env == lib.PROD {
			mtsClient.Domain = api.MtsVpcDomain
		}
		api.StsMtsClient[k] = &api.AliMtsClient{
			Client: mtsClient,
			Sts:    stsData,
		}
	}
}
func RemoteAliClient() {
	go func() {
		getOssClientRemote()
		getMtsClientRemote()
		for {
			select { //每5分钟更新一次
			case <-time.After(time.Minute * 5):
				getOssClientRemote()
				getMtsClientRemote()
			}
		}
	}()
}
