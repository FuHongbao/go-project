package main

/*
import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

type ReqGetMultiUploadInfo struct {
	Kind    	int    	`json:"type"`
	QeTag 		string 	`json:"qetag"`
	FileSize 	int64	`json:"size"`
}
type Chunk struct {
	Number 			int   	`json:"chunk_num"`
	Offset 			int64 	`json:"offset"`
	Size   			int64 	`json:"chunk_size"`
	Ready  			bool  	`json:"ready"`
}

type MultiUploadInfo struct {
	Chunks    	[]Chunk 	`json:"chunks"`
	ChunkCnt  	int  	  	`json:"chunk_cnt"`
	Key       	string  	`json:"key"`
	UploadID  	string  	`json:"upload_id"`
	Parts       []PartData 	`json:"parts"`
}
type RespUploadConf struct {
	Ret  int 	`json:"ret"`
	Data MultiUploadInfo `json:"data"`
}

func GetUploadConf(url , qeTag string, fileSize int64) (conf MultiUploadInfo, err error) {
	client := &http.Client{}
	data := &ReqGetMultiUploadInfo{Kind:6, QeTag:qeTag, FileSize:fileSize}
	bytesData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bytesData))
	if err != nil {
		fmt.Println(err)
		return
	}
	confResp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	if confResp == nil {
		fmt.Println("resp is nil")
		return
	}
	defer confResp.Body.Close()
	result, err := ioutil.ReadAll(confResp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("resp unmarshal", confResp)
	var uploadConf RespUploadConf
	err = json.Unmarshal(result, &uploadConf)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(result))
	conf = uploadConf.Data
	return
}

type PartData struct {
	Etag    string 	`json:"etag"`
	Num     int 	`json:"num"`
}

type ReqMultiPartAuth struct {
	Key      string `json:"key"`
	UploadID string `json:"upload_id"`
	ChunkNum int    `json:"chunk_num"`
	Md5Value string `json:"md5_value"`
}

type RespMultiPartAuth struct {
	Url           string `json:"url"`
	Authorization string `json:"authorization"`
	Date          string `json:"date"`
	Token 		  string `json:"x-oss-security-token"`
	Host          string `json:"host"`
	Method        string `json:"method"`
	ContentType   string `json:"Content-Type"`
	ExpireSec     int    `json:"expire_sec"`
}
type RespAuthorConf struct {
	Ret  int 	`json:"ret"`
	Data RespMultiPartAuth `json:"data"`
}

func GetAuthorization(key, uploadID, Md5Value string, ChunkNum int) (resp RespMultiPartAuth, err error) {
	url := "http://test-kapi.xiaoniangao.cn/resources_center/authorize/get_multi_author"
	client := &http.Client{}
	data := &ReqMultiPartAuth{
		Key:      key,
		UploadID: uploadID,
		ChunkNum: ChunkNum,
		Md5Value: Md5Value,
	}
	bytesData, _ := json.Marshal(data)
	req, nerr := http.NewRequest("POST", url, bytes.NewReader(bytesData))
	if nerr != nil {
		fmt.Println(nerr)
		err = nerr
		return
	}
	authResp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	if authResp == nil {
		fmt.Println("authResp is nil")
		return
	}
	defer authResp.Body.Close()
	result, err := ioutil.ReadAll(authResp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	var respData RespAuthorConf
	err = json.Unmarshal(result, &respData)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(result))
	resp = respData.Data
	return
}
func UploadPartsSection(conf MultiUploadInfo, cnt int, fd *os.File) (parts []PartData, err error) {
	client := &http.Client{}
	var Part PartData
	for _,chunk := range conf.Chunks {
		cnt--
		if cnt <= 0 {
			break
		}
		fd.Seek(chunk.Offset, os.SEEK_SET)
		buf := make([]byte, chunk.Size)
		_, nerr := fd.Read(buf)
		if nerr != nil {
			err = nerr
			fmt.Println(nerr)
			return
		}
		md5Ctx := md5.New()
		md5Ctx.Write(buf)
		md5Value := md5Ctx.Sum(nil)
		md5Str := base64.StdEncoding.EncodeToString(md5Value)
		authData, nerr := GetAuthorization(conf.Key, conf.UploadID, md5Str, chunk.Number)
		if nerr != nil {
			fmt.Println(nerr)
			err = nerr
			return
		}
		req, nerr := http.NewRequest(authData.Method, authData.Url, bytes.NewReader(buf))
		if nerr != nil {
			fmt.Println(nerr)
			err = nerr
			return
		}
		req.Header.Set("Content-MD5", md5Str)
		req.Header.Set("Content-Type", authData.ContentType)
		req.Header.Set("Host", authData.Host)
		req.Header.Set("Content-Length", strconv.FormatInt(chunk.Size, 10))
		req.Header.Set("Date", authData.Date)
		req.Header.Set("Authorization", authData.Authorization)
		req.Header.Add("x-oss-security-token", authData.Token)
		req.Proto = "HTTP/1.1"
		req.ProtoMajor = 1
		req.ProtoMinor = 1
		resp, nerr1 := client.Do(req)
		if nerr1 != nil {
			err = nerr1
			fmt.Println(nerr1)
			return
		}
		result, nerr2 := ioutil.ReadAll(resp.Body)
		if nerr2 != nil {
			err = nerr2
			fmt.Println(nerr2)
			return
		}
		fmt.Println(string(result))
		Part.Num = chunk.Number
		Part.Etag = resp.Header.Get("Etag")
		parts = append(parts, Part)
	}
	return
}

func UploadParts(conf MultiUploadInfo, fd *os.File) (parts []PartData, err error) {

	client := &http.Client{}
	var Part PartData
	for _,chunk := range conf.Chunks {
		if chunk.Ready == true {
			continue
		}
		fd.Seek(chunk.Offset, os.SEEK_SET)
		buf := make([]byte, chunk.Size)
		_, nerr := fd.Read(buf)
		if nerr != nil {
			err = nerr
			fmt.Println(nerr)
			return
		}
		md5Ctx := md5.New()
		md5Ctx.Write(buf)
		md5Value := md5Ctx.Sum(nil)
		md5Str := base64.StdEncoding.EncodeToString(md5Value)
		authData, nerr := GetAuthorization(conf.Key, conf.UploadID, md5Str, chunk.Number)
		if nerr != nil {
			fmt.Println(nerr)
			err = nerr
			return
		}
		req, nerr := http.NewRequest(authData.Method, authData.Url, bytes.NewReader(buf))
		if nerr != nil {
			fmt.Println(nerr)
			err = nerr
			return
		}
		req.Header.Set("Content-MD5", md5Str)
		req.Header.Set("Content-Type", authData.ContentType)
		req.Header.Set("Host", authData.Host)
		req.Header.Set("Content-Length", strconv.FormatInt(chunk.Size, 10))
		req.Header.Set("Date", authData.Date)
		req.Header.Set("Authorization", authData.Authorization)
		req.Header.Add("x-oss-security-token", authData.Token)
		req.Proto = "HTTP/1.1"
		req.ProtoMajor = 1
		req.ProtoMinor = 1
		resp, nerr1 := client.Do(req)
		if nerr1 != nil {
			err = nerr1
			fmt.Println(nerr1)
			return
		}
		result, nerr2 := ioutil.ReadAll(resp.Body)
		if nerr2 != nil {
			err = nerr2
			fmt.Println(nerr2)
			return
		}
		fmt.Println(string(result))
		Part.Num = chunk.Number
		Part.Etag = resp.Header.Get("Etag")
		parts = append(parts, Part)
	}
	return
}

type ReqCheckMultiUpload struct {
	Kind    int    `json:"type"`
	QeTag   string `json:"qetag"`
	Product int    `json:"prod"` //产品 1 xng 2 xbd 3 tia
	Project string `json:"proj"` //ma app ...
	Key 	string `json:"key"`
	UploadID string     `json:"upload_id"`
	ContentType string 	`json:"Content-Type"`
	Parts   []PartData  `json:"parts"`
}
type ResourceInfo struct {
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
}

type RespCheckMultiUpload struct {
	Status int          `json:"status"`
	Info   ResourceInfo `json:"info"`
}
type RespCheckData struct {
	Ret  int 	`json:"ret"`
	Data RespCheckMultiUpload `json:"data"`
}

func CheckStatus(req ReqCheckMultiUpload, url string)(Data RespCheckMultiUpload, err error) {
	client := &http.Client{}
	bytesCheckData, _ := json.Marshal(req)
	checkReq, err := http.NewRequest("POST", url, bytes.NewBuffer(bytesCheckData))
	if err != nil {
		fmt.Println(err)
		return
	}
	resp, err := client.Do(checkReq)
	if err != nil {
		fmt.Println(err)
		return
	}
	if resp == nil {
		fmt.Println("resp is nil")
		return
	}
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	var respData RespCheckData
	err = json.Unmarshal(result, &respData)
	if err != nil {
		fmt.Println(err)
		return
	}
	Data = respData.Data
	fmt.Println(string(result))
	return
}


func AppendParts(AllParts, parts []PartData) (retParts []PartData) {
	for _, part := range parts {
		AllParts = append(AllParts, part)
	}
	retParts = AllParts
	return
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
	qetag := "otdgasfffiopokfyy"   //测试前请更新qetag
	LocalFileName := "D:\\xng_project\\3099468768.mp4"
	ConfUrl := "http://test-kapi.xiaoniangao.cn/resources_center/resource/multi_upload_config"
	CheckUrl := "http://test-kapi.xiaoniangao.cn/resources_center/resource/check_multi_upload_result"
	fd, err := os.Open(LocalFileName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer fd.Close()
	stat, err := fd.Stat()
	if err != nil {
		fmt.Println(err)
		return
	}
	fileSize := stat.Size()
	contentType, err := GetFileContentType(fd)
	if err != nil {
		fmt.Println(err)
		return
	}
	if contentType == "" {
		fmt.Println("content_type is nil")
		return
	}
	//fmt.Println(contentType)
	uploadConf, err := GetUploadConf(ConfUrl, qetag, fileSize)
	if err != nil {
		fmt.Println(err)
		return
	}
	var AllParts []PartData
	parts, err := UploadPartsSection(uploadConf, 3, fd)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(parts)

	resumeConf, err := GetUploadConf(ConfUrl, qetag, fileSize)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("resume data: ", resumeConf)
	AllParts = AppendParts(AllParts, resumeConf.Parts)

	parts, err = UploadParts(resumeConf, fd)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(parts)
	AllParts = AppendParts(AllParts, parts)

	CheckReqData := ReqCheckMultiUpload{
		Kind:    6,
		QeTag:   qetag,
		Product: 1,
		Project: "11",
		Key:     uploadConf.Key,
		UploadID: uploadConf.UploadID,
		ContentType:contentType,
		Parts:   AllParts,
	}

	_, err = CheckStatus(CheckReqData, CheckUrl)
	if err != nil {
		fmt.Println(err)
		return
	}
}

*/
