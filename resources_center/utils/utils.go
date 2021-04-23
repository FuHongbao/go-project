package utils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
)

func GetMilliTime() int64 {
	return time.Now().UnixNano() / 1e6
}

var indexPipe int

func init() {
	indexPipe = 0
}

type ResCallBack struct {
	Key      string `json:"key"`
	Path     string `json:"path"`
	Kind     int    `json:"type"`
	Product  int    `json:"prod"`
	Project  string `json:"proj"`
	JobType  string `json:"job_type"`
	UserData string `json:"user_data"`
}

func SetMNSCallBackData(key, path string, kind, product int, project, action, userData string) (string, error) {
	data := ResCallBack{
		Key:      key,
		Kind:     kind,
		JobType:  action,
		Product:  product,
		Project:  project,
		Path:     path,
		UserData: userData,
	}
	dataStr, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	finStr := base64.StdEncoding.EncodeToString(dataStr)
	return finStr, nil
}

func GetMNSCallBackData(ctx context.Context, data string) (userData *ResCallBack, err error) {
	dataBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		xlog.ErrorC(ctx, "GetMNSCallBackData.DecodeString failed, data:[%s], err:[%v]", data, err)
		return
	}
	err = json.Unmarshal(dataBytes, &userData)
	if err != nil {
		return
	}
	return
}
func GetMTSPipeId(env string) string {
	if env == lib.PROD {
		return api.TransCodePipelineId
	}
	return api.TransCodePipeIdForTest
}

func GetMediaInfoPipe(env string) string {
	if env == lib.PROD {
		return api.TransCodePipelineId
	}
	return api.TransCodePipeIdForTest
}

func GetQetagByResType(ty int, qeTag string) string {
	switch ty {
	case api.ResourceTypeAlbum:
		return fmt.Sprintf("%d%s", api.ResourceTypeAlbum, qeTag)
	case api.ResourceTypeLive:
		return fmt.Sprintf("%d%s", api.ResourceTypeLive, qeTag)
	case api.ResourceTypeGuideVideo:
		return fmt.Sprintf("%d%s", api.ResourceTypeGuideVideo, qeTag)
	}
	return qeTag
}

func GetMediaInfoTopic(proj string, prod int, userService string) (topic, tag string, err error) {
	if userService != "" {
		topic = fmt.Sprintf("upload_topic_info_%s", userService)
		return
	}
	topic, ok := api.MqInfoTopicMap[prod]
	if !ok {
		err = errors.New(fmt.Sprintf("unknow type of product:%d", prod))
		return
	}
	tag = fmt.Sprintf("src_%s", proj)
	return
}

func GetVideoTransTopic(userService string) (topic, tag string) {
	return fmt.Sprintf("trans_topic_result_%s", userService), ""
}

func GetVideoTransTemplate(tplID string) string {
	switch tplID {
	case api.TransTemplate720PMP4:
		return api.TransCodeTemplateId
	}
	return ""
}
func GetOutputByKind(kind int, id int64) string {
	switch kind {
	case api.ResourceTypeGuideVideo:
		return fmt.Sprintf("%d.mp4", id)
	default:
		return strconv.FormatInt(id, 10)
	}
}
