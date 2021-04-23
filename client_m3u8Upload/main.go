package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
)

type ReqGetUploadInfo struct {
	Kind    int    `json:"type"`
	QeTag   string `json:"qetag"`
	Product int    `json:"prod"`    //产品 1 xng 2 xbd 3 tia
	Project string `json:"proj"`    //ma app ...
	NoBack  bool   `json:"no_back"` //是否取消回调，默认值为false，进行回调
	NoMq    bool   `json:"no_mq"`
}

type UploadInfo struct {
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

type RespUploadInfo struct {
	Host             string                  `json:"host"`
	InternalHost     string                  `json:"internal_host"`
	UploadInfo       UploadInfo          	 `json:"upload_info"`
	UploadCustomInfo CallbackCustomParam 	 `json:"upload_custom_info"`
	ID               string                  `json:"id"`
	ExpireSec        int                     `json:"expire_sec"`   //这个字段目前显示的时间不准确，不需要在意该字段的值
}

type RespUploadInfoData struct {
	Ret   	int 	`json:"ret"`
	Data 	*RespUploadInfo  `json:"data"`
}

func GetUploadConfig(url string, kind int, qetag string, prod int, proj string) (config *RespUploadInfo, err error) {
	client := &http.Client{}
	data := &ReqGetUploadInfo{
		Kind:    kind,
		QeTag:   qetag,
		Product: prod,
		Project: proj,
		NoBack:  false,
		NoMq:    false,
	}
	bytesData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bytesData))
	if err != nil {
		fmt.Println(err)
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	if resp == nil {
		err = errors.New("GetUploadConfig resp is nil")
		return
	}
	defer resp.Body.Close()
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var uploadConf RespUploadInfoData
	err = json.Unmarshal(result, &uploadConf)
	if err != nil {
		return
	}
	if uploadConf.Ret != 1 {
		err = errors.New(fmt.Sprintf("GetUploadConfig ret: [%d], data: [%v]", uploadConf.Ret, uploadConf.Data))
		return
	}
	config = uploadConf.Data
	return
}

func UploadFile(url string, params map[string]string,  fileName string) ([]byte, error) {
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
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

func GetFileContentType(out *os.File) (string, error) {

	buffer := make([]byte, 512)
	_, err := out.Read(buffer)
	if err != nil {
		return "", err
	}
	contentType := http.DetectContentType(buffer)

	return contentType, nil
}


func main() {
	kind := 12
	qetag := "otdgawfwzfeillefdxfwfd=ffwyool"   //每次测试前请更新qetag
	proj := "11"  //上传来源：小程序
	prod := 1	 //所属产品：1：小年糕
	LocalFileName := "D:\\xng_project\\m3u8\\index.m3u8"
	ConfUrl := "http://test-kapi.xiaoniangao.cn/resources_center/resource/get_upload_info"
	fd, err := os.Open(LocalFileName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer fd.Close()
	contentType, err := GetFileContentType(fd)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(contentType)
	config, err := GetUploadConfig(ConfUrl, kind, qetag, prod, proj)
	if err != nil {
		fmt.Println("failed to get upload config : ", err)
		return
	}
	LocalFileName = fmt.Sprintf("D:\\xng_project\\m3u8\\test\\index.m3u8")
	param := map[string]string{}
	param["key"] = fmt.Sprintf("3815712/index.m3u8")
	param["policy"] =config.UploadInfo.Policy
	param["OSSAccessKeyId"] =config.UploadInfo.AccessKey
	param["success_action_status"] =config.UploadInfo.SuccessActionStatus
	//param["callback"] = config.UploadInfo.Callback
	param["signature"] = config.UploadInfo.Signature
	param["x-oss-security-token"] = config.UploadInfo.SecurityToken
	bts, _ := json.Marshal(config.UploadCustomInfo)
	myvar := base64.StdEncoding.EncodeToString(bts)
	param["x:my_var"] = myvar
	fmt.Println(param["key"])
	body, err := UploadFile(config.Host, param, LocalFileName)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(body))
	return
	for i := 0; i <= 0; i++ {
		LocalFileName = fmt.Sprintf("D:\\xng_project\\m3u8\\test\\video-0000%d.ts", i)
		param := map[string]string{}
		param["key"] = fmt.Sprintf("3815712/video-0000%d.ts", i)
		param["policy"] =config.UploadInfo.Policy
		param["OSSAccessKeyId"] =config.UploadInfo.AccessKey
		param["success_action_status"] =config.UploadInfo.SuccessActionStatus
		//param["callback"] = config.UploadInfo.Callback
		param["signature"] = config.UploadInfo.Signature
		param["x-oss-security-token"] = config.UploadInfo.SecurityToken
		bts, _ := json.Marshal(config.UploadCustomInfo)
		myvar := base64.StdEncoding.EncodeToString(bts)
		param["x:my_var"] = myvar
		fmt.Println(param["key"])
		body, err := UploadFile(config.Host, param, LocalFileName)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(string(body))
	}
}
