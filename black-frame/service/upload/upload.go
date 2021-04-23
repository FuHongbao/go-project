package upload

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/api"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/conf"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/util/common"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/util/net"
	"xgit.xiaoniangao.cn/xngo/service/ids_api"
)

const (
	ServiceName       = "resources-center"
	ServiceUploadPath = "resource/get_upload_info"
	ReTryTimes        = 3
)

type ReqCommUpload struct {
	Kind    int    `json:"type"`
	QeTag   string `json:"qetag"`
	Product int    `json:"prod"`    //产品 1 xng 2 xbd 3 tia
	Project string `json:"proj"`    //ma app ...
	NoBack  bool   `json:"no_back"` //是否取消回调，默认值为false，进行回调
	NoMq    bool   `json:"no_mq"`
}
type CommUploadConf struct {
	Signature           string `json:"Signature"`
	Policy              string `json:"policy"`
	Callback            string `json:"Callback"`
	Key                 string `json:"key"` //字段变为id，后期类型将变为string类型
	AccessKey           string `json:"OSSAccessKeyId"`
	SuccessActionStatus string `json:"success_action_status"`
	SecurityToken       string `json:"x-oss-security-token"`
}

type CallbackCustomParam struct {
	Kind    int    `json:"type"`
	QeTag   string `json:"qetag"`
	Product int    `json:"prod"` //产品 1 xng 2 xbd 3 tia
	Project string `json:"proj"` //ma app ...
	NoMq    int    `json:"no_mq"`
}
type RespCommUpload struct {
	Host             string              `json:"host"`
	InternalHost     string              `json:"internal_host"`
	UploadInfo       CommUploadConf      `json:"upload_info"`
	UploadCustomInfo CallbackCustomParam `json:"upload_custom_info"`
	ID               string              `json:"id"`
	ExpireSec        int                 `json:"expire_sec"` //这个字段目前显示的时间不准确，不需要在意该字段的值
}
type RespUploadInfoData struct {
	Ret  int             `json:"ret"`
	Data *RespCommUpload `json:"data"`
}

func GetCommUploadConfig(ctx context.Context, prod int, proj string, resType int, qeTag string) (config *RespCommUpload, err error) {
	req := ReqCommUpload{
		Kind:    resType,
		QeTag:   qeTag,
		Product: prod,
		Project: proj,
		NoBack:  false,
		NoMq:    true,
	}
	resp := RespUploadInfoData{}
	if conf.Env == lib.PROD {
		err = net.XngServiceCallPostWithRetry(ctx, &net.Consul{}, ServiceName, ServiceUploadPath, req, &resp, time.Second*2, 1)
		if err != nil {
			return
		}
	} else {
		err = net.Post(ctx, api.CommUploadConfigURL, time.Second*2, req, &resp)
		if err != nil {
			return
		}
	}
	config = resp.Data
	return
}

func DoUploadFile(url string, params map[string]string, fileName string) ([]byte, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	body := new(bytes.Buffer)

	writer := multipart.NewWriter(body)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}

	formFile, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(formFile, file)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func ResUploadToOSS(ctx context.Context, filePath string, config *RespCommUpload) (err error) {
	//config, err := GetCommUploadConfig(ctx, prod, proj, resType, qeTag)
	//if err != nil {
	//	return
	//}
	param := map[string]string{}
	param["key"] = config.UploadInfo.Key
	param["policy"] = config.UploadInfo.Policy
	param["OSSAccessKeyId"] = config.UploadInfo.AccessKey
	param["success_action_status"] = config.UploadInfo.SuccessActionStatus
	param["callback"] = config.UploadInfo.Callback
	param["signature"] = config.UploadInfo.Signature
	param["x-oss-security-token"] = config.UploadInfo.SecurityToken
	bts, _ := json.Marshal(config.UploadCustomInfo)
	myvar := base64.StdEncoding.EncodeToString(bts)
	param["x:my_var"] = myvar
	resp, err := DoUploadFile(config.Host, param, filePath)
	if err != nil {
		return
	}
	xlog.DebugC(ctx, "ResUploadToOSS.DoUploadFile resp:[%s]", string(resp))
	return
}

func ResUploadByResourceCenter(ctx context.Context, filePath string, prod int, proj string, key string, resType int) (err error) {
	qeTag, err := common.GetResourceQeTag(ctx, filePath)
	if err != nil {
		return
	}
	config, err := GetCommUploadConfig(ctx, prod, proj, resType, qeTag)
	if err != nil {
		return
	}
	config.UploadInfo.Key = key
	err = ResUploadToOSS(ctx, filePath, config)
	if err != nil {
		return
	}
	xlog.DebugC(ctx, "ResUploadByResourceCenter success, filename:[%s], oss keyName:[%s], resType:[%d]", filePath, key, resType)
	return
}

func GetDistributedId(ch chan int64) {
	var id int64
	var ok bool
	for i := 0; i < ReTryTimes; i++ {
		resp, err := ids_api.GetNewIds(conf.C.Addrs.Ids, "xng-res") //表名为res，主键为res
		if err != nil {
			xlog.Error("get new distributed id error:%v", err)
			continue
		}
		if resp == nil {
			xlog.Error("get new distributed id error, resp nil")
			continue
		}
		id, ok = resp.Data["id"]
		if ok == false {
			xlog.Error("get new distributed id error, resp data nil, resp:%v", resp.Data)
			id = 0
			continue
		}
		break
	}
	ch <- id
}
