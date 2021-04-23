package trans

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/mts"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/resource"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils"
)

func VideoMp4Trans(ctx context.Context, input string, kind int, prod int, proj, tplID string, userData string, serviceName string) (status int, output string, jobId string, err error) {
	ch := make(chan int64, 1)
	go resource.GetDistributedId(ch)
	stsName := resource.GetMtsStsName(kind)
	mtsInfo, err := resource.GetAliMtsClient(ctx, stsName)
	if err != nil {
		return
	}
	data := api.MultiVideoUserData{
		UserData:    userData,
		UserService: serviceName,
		OldId:       input,
		TplId:       tplID,
	}
	userByte, nerr := json.Marshal(data)
	if nerr != nil {
		err = nerr
		return
	}
	newId := <-ch
	if newId <= 0 {
		err = errors.New("VideoMp4Trans.GetDistributedId failed, id is nil")
		return
	}
	output = utils.GetOutputByKind(kind, newId)
	callbackData, err := utils.SetMNSCallBackData(output, "", kind, prod, proj, api.MTSJobTypeTransVideo, string(userByte))
	if err != nil {
		return
	}
	endPoint := mtsInfo.Sts.Endpoint
	location := endPoint[:len(endPoint)-13]
	request := mts.CreateSubmitJobsRequest()
	request.Scheme = "http"
	request.Input = fmt.Sprintf("{\"Bucket\":\"%s\",\"Location\":\"%s\",\"Object\":\"%s\"}", mtsInfo.Sts.Bucket, location, input)
	request.Outputs = fmt.Sprintf("[{\"OutputObject\":\"%s\",\"TemplateId\":\"%s\",\"UserData\":\"%s\"}]", output, utils.GetVideoTransTemplate(tplID), callbackData)
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
