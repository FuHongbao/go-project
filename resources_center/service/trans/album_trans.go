package trans

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/mts"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/resource"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils"
)

type AlbumTransCallBack struct {
	Key      string `json:"key"`
	Kind     int    `json:"type"`
	Product  int    `json:"prod"`
	Project  string `json:"proj"`
	Env      string `json:"env"`
	JobType  string `json:"job_type"`
	UserData string `json:"user_data"`
}

func getAlbumTransCallbackParam(key string, kind int, product int, project string, jobType string, userData string) (string, error) {
	data := AlbumTransCallBack{
		Key:      key,
		Kind:     kind,
		JobType:  jobType,
		Product:  product,
		Project:  project,
		Env:      conf.Env,
		UserData: userData,
	}
	dataStr, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	finStr := base64.StdEncoding.EncodeToString(dataStr)
	return finStr, nil
}

func AlbumM3u8Trans(ctx context.Context, key string, kind int, prod int, proj string, userData string) (status int, jobId string, err error) {
	stsName := resource.GetMtsStsName(api.ResourceTypeAlbum)
	mtsInfo, err := resource.GetAliMtsClient(ctx, stsName)
	if err != nil {
		return
	}
	//callbackData, err := getAlbumTransCallbackParam(key, kind, prod, proj, api.MTSJobTypeTransVideo, userData)
	callbackData, err := utils.SetMNSCallBackData(key, "", kind, prod, proj, api.MTSJobTypeTransVideo, userData)
	if err != nil {
		return
	}
	fileName := key + "/index_0.m3u8"
	endPoint := mtsInfo.Sts.Endpoint
	location := endPoint[:len(endPoint)-13]
	request := mts.CreateSubmitJobsRequest()
	request.Scheme = "http"
	request.Input = fmt.Sprintf("{\"Bucket\":\"%s\",\"Location\":\"%s\",\"Object\":\"%s\"}", mtsInfo.Sts.Bucket, location, fileName)
	request.Outputs = fmt.Sprintf("[{\"OutputObject\":\"%s\",\"TemplateId\":\"%s\",\"UserData\":\"%s\"}]", key, api.TransCodeTemplateId, callbackData)
	request.OutputBucket = mtsInfo.Sts.Bucket
	request.PipelineId = utils.GetMTSPipeId(conf.Env)
	request.OutputLocation = location
	resp, err := mtsInfo.Client.SubmitJobs(request)
	if err != nil {
		fmt.Println(err)
		return
	}
	if resp.JobResultList.JobResult[0].Success == true {
		status = 1
		jobId = resp.JobResultList.JobResult[0].Job.JobId
		return
	} else {
		xlog.DebugC(ctx, "AlbumM3u8Trans.SubmitJobs failed, resp:[%v]", resp)
	}
	return
}
