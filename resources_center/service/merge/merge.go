package merge

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/mts"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"strings"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/sts"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils"
)

type VideoParam struct {
	Height int `json:"Height"`
	Width  int `json:"Width"`
}
type ListData struct {
	URL string `json:"MergeURL"`
}
type OutputData struct {
	Key      string     `json:"OutputObject"`
	Template string     `json:"TemplateId"`
	Video    VideoParam `json:"Video"`
	List     []ListData `json:"MergeList,omitempty"`
	Url      string     `json:"MergeConfigUrl,omitempty"`
	UserData string     `json:"UserData"`
}
type OutputDataV2 struct {
	Key      string     `json:"OutputObject"`
	Template string     `json:"TemplateId"`
	Video    VideoParam `json:"Video"`
	Url      string     `json:"MergeConfigUrl,omitempty"`
	UserData string     `json:"UserData"`
}
type ResMergeCallBack struct {
	Key      string `json:"key"`
	Path     string `json:"path"`
	Kind     int    `json:"type"`
	Product  int    `json:"prod"`
	Project  string `json:"proj"`
	Env      string `json:"env"`
	JobType  string `json:"job_type"`
	UserData string `json:"user_data"`
}

func getVideoMergeCallbackParam(key, path string, kind int, product int, project string, jobType string, userData string) (string, error) {
	data := ResMergeCallBack{
		Key:      key,
		Kind:     kind,
		JobType:  jobType,
		Product:  product,
		Project:  project,
		Path:     path,
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

func HandleVideoMerge(ctx context.Context, ids []string, w, h int, stsData *sts.UploadToken, key, path string, kind, product int, project string, userData string) (jobId string, status int, err error) {
	endPoint := stsData.Endpoint
	//if conf.Env == lib.PROD {
	//	endPoint = stsData.EndpointInternal
	//}
	regionID := endPoint[4 : len(endPoint)-13]
	client, err := mts.NewClientWithStsToken(regionID, stsData.AccessKey, stsData.SecretKey, stsData.SecurityToken)
	if err != nil {
		return
	}
	location := endPoint[:len(endPoint)-13]
	request := mts.CreateSubmitJobsRequest()
	request.Input = fmt.Sprintf("{\"Bucket\":\"%s\",\"Location\":\"%s\",\"Object\":\"%s\"}", stsData.Bucket, location, ids[0])
	var output OutputData
	output.Key = path + key
	output.Template = api.TransCodeTemplateId
	output.Video.Height = h
	output.Video.Width = w
	for i := 1; i < len(ids); i++ {
		mergeURL := fmt.Sprintf("http://%s.%s/%s", stsData.Bucket, endPoint, ids[i])
		mergeOne := ListData{URL: mergeURL}
		output.List = append(output.List, mergeOne)
	}
	//output.UserData, err = getVideoMergeCallbackParam(key, path, kind, product, project, api.MTSJobTypeMergeVideo, userData)
	output.UserData, err = utils.SetMNSCallBackData(key, path, kind, product, project, api.MTSJobTypeMergeVideo, userData)
	if err != nil {
		return
	}
	var outputList []OutputData
	outputList = append(outputList, output)
	outputBytes, err := json.Marshal(outputList)
	request.Outputs = string(outputBytes)
	request.OutputBucket = stsData.Bucket
	request.OutputLocation = location
	request.PipelineId = api.MergeVideoPipeLineId
	resp, err := client.SubmitJobs(request)
	if err != nil {
		return
	}
	if resp == nil {
		xlog.ErrorC(ctx, "HandleVideoMerge.SubmitJobs failed resp is nil")
		err = errors.New("SubmitJobs resp is nil")
		return
	}
	if resp.JobResultList.JobResult[0].Success == true {
		jobId = resp.JobResultList.JobResult[0].Job.JobId
		status = 1
		return
	} else if resp.JobResultList.JobResult[0].Success == false {
		xlog.DebugC(ctx, "HandleVideoMerge.SubmitJobs failed msg:[%v]", resp.JobResultList.JobResult[0].Message)
	}
	return
}
func HandleLargeCntVideoMerge(ctx context.Context, urlPath, firstPart string, w, h int, stsData *sts.UploadToken, key, path string, kind, product int, project string, userData string) (jobId string, status int, err error) {
	endPoint := stsData.Endpoint
	regionID := endPoint[4 : len(endPoint)-13]
	client, err := mts.NewClientWithStsToken(regionID, stsData.AccessKey, stsData.SecretKey, stsData.SecurityToken)
	if err != nil {
		return
	}
	location := endPoint[:len(endPoint)-13]
	request := mts.CreateSubmitJobsRequest()
	request.Input = fmt.Sprintf("{\"Bucket\":\"%s\",\"Location\":\"%s\",\"Object\":\"%s\"}", stsData.Bucket, location, firstPart)
	var output OutputDataV2
	output.Key = path + key
	output.Template = api.TransCodeTemplateId
	output.Video.Height = h
	output.Video.Width = w
	output.Url = urlPath
	//output.UserData, err = getVideoMergeCallbackParam(key, path, kind, product, project, api.MTSJobTypeMergeVideo, userData)
	output.UserData, err = utils.SetMNSCallBackData(key, path, kind, product, project, api.MTSJobTypeMergeVideo, userData)
	if err != nil {
		return
	}
	var outputList []OutputDataV2
	outputList = append(outputList, output)
	outputBytes, err := json.Marshal(outputList)
	request.Outputs = string(outputBytes)
	request.OutputBucket = stsData.Bucket
	request.OutputLocation = location
	request.PipelineId = utils.GetMTSPipeId(conf.Env)
	resp, err := client.SubmitJobs(request)
	if err != nil {
		fmt.Println(err)
		return
	}
	if resp.JobResultList.JobResult[0].Success == true {
		status = 1
		jobId = resp.JobResultList.JobResult[0].Job.JobId
		return
	}
	return
}
func getMergeConf(key, env string) string {
	return fmt.Sprintf("MergeConf/%s/%s_conf", env, key)
}

type VideoMergeConfURL struct {
	List []ListData `json:"MergeList"`
}

func getVideoMergeConfValue(ids []string, bucket, endpoint string) (value string, err error) {
	var mergeList VideoMergeConfURL
	for _, id := range ids {
		url := fmt.Sprintf("http://%s.%s/%s", bucket, endpoint, id)
		listData := ListData{URL: url}
		mergeList.List = append(mergeList.List, listData)
	}
	bts, err := json.Marshal(mergeList)
	if err != nil {
		return
	}
	value = string(bts)
	return
}
func UploadMergeVideoConf(ctx context.Context, stsData *sts.UploadToken, ids []string, bucket, key string, env string) (confURL string, err error) {
	endPoint := stsData.EndpointInternal
	if conf.Env != lib.PROD {
		endPoint = stsData.Endpoint
	}
	client, err := oss.New(endPoint, stsData.AccessKey, stsData.SecretKey, oss.SecurityToken(stsData.SecurityToken))
	if err != nil {
		return
	}
	ossBucket, err := client.Bucket(bucket)
	if err != nil {
		return
	}
	objectName := getMergeConf(key, env)
	objectValue, err := getVideoMergeConfValue(ids, bucket, endPoint)
	if err != nil {
		return
	}
	err = ossBucket.PutObject(objectName, strings.NewReader(objectValue))
	if err != nil {
		return
	}
	err = ossBucket.SetObjectMeta(objectName, oss.ContentType(api.ContentTypeJson))
	if err != nil {
		xlog.ErrorC(ctx, "file:%v set Content-Type:%v error, error:%v", objectName, api.ContentTypeTxt, err)
		return
	}
	confURL = fmt.Sprintf("http://%s.%s/%s", bucket, endPoint, objectName)
	return
}
