package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)


const (
	ResIDsFile      =  "./ids.txt"
	LogPathError    =  "./error.log"
	LogPathInfo     =  "./info.log"
	FailedIds       =  "./fail_ids.txt"
	OkIds           =  "./ok_ids.txt"
	EnvProd         =  "prod"
)

type ReqVideoUrl struct {
	ID string `json:"id"`
}

type RespURL struct {
	URL         string `json:"url"`
	URLInternal string `json:"url_internal"`
}
type RespVideoUrlData struct {
	URLs map[string]RespURL `json:"urls"`
}
type RespVideoUrl struct {
	Ret  int              `json:"ret"`
	Data RespVideoUrlData `json:"data"`
}
func getURLReqByEnv(env string) string {
	if env == EnvProd {
		return "https://kapi.xiaoniangao.cn/resources_center/url/album"
	} else {
		return "http://192.168.11.50:8987/url/album"
	}
}
func GetVideoUrlByID(key string, env string) (url string, err error) {
	reqUrl := getURLReqByEnv(env)
	var request []ReqVideoUrl
	param := ReqVideoUrl{ID: key}
	request = append(request, param)
	response := RespVideoUrl{}
	bts, _ := json.Marshal(request)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewReader(bts))
	if err != nil {
		return
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	if err != nil {
		return
	}
	//if env == EnvProd {
	//	url = response.Data.URLs[key].URLInternal
	//} else {
	//	url = response.Data.URLs[key].URL
	//}
	url = response.Data.URLs[key].URL
	writeLogInfo(fmt.Sprintf("GetVideoUrlByID success get url:[%s]", url))
	return
}

func DownLoadResource(url string, dir string, key string) (path string, err error) {
	response, err := http.Get(url)
	if err != nil {
		return
	}
	defer response.Body.Close()
	path = dir + key + ".mp4"
	fp, err := os.Create(path)
	if err != nil {
		return
	}
	defer fp.Close()
	_, err = io.Copy(fp, response.Body)
	if err != nil {
		return
	}
	return
}


//type MediaFormat struct {
//	Format     string `json:"format_name"`
//	DuString   string `json:"duration"`
//	SizeString string `json:"size"`
//}

type MediaStream struct {
	Index     int    `json:"index"`
	CodeName  string `json:"codec_name"`
	CodeType  string `json:"codec_type"`
	FrameRate string `json:"r_frame_rate"`
	Du        string `json:"duration"`
}
type MediaInfoByFFmpeg struct {
	//Format MediaFormat   `json:"format"`
	Stream 		[]MediaStream 	`json:"streams"`
	Rate   		string 			`json:"-"`
	Du       	float64 		`json:"-"`
}

func GetMediaInfoByFFMPEG(path string) (mediaInfo MediaInfoByFFmpeg, err error) {
	out, err := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_streams", "-i", path).Output()
	if err != nil {
		return
	}
	if out == nil {
		err = errors.New("GetMediaInfoByFFMPEG failed out is nil")
		return
	}
	err = json.Unmarshal(out, &mediaInfo)
	if err != nil {
		return
	}
	if mediaInfo.Stream == nil {
		err = errors.New("GetMediaInfoByFFMPEG failed stream is nil")
		return
	}
	for _, stream := range mediaInfo.Stream {
		if stream.CodeType == "video" {
			mediaInfo.Rate = stream.FrameRate
			mediaInfo.Du, _ = strconv.ParseFloat(stream.Du, 10)
			return
		}
	}
	return
}
func getVideoRate(rate string) string {
	switch rate {
	case "20/1":
		return "100"
	case "16/1":
		return "80"
	case "24/1":
		return "120"
	case "25/1":
		return "125"
	case "30/1":
		return "150"
	}
	return ""
}

func getM3u8DestPath(dir string) string {
	return dir + "index.m3u8"
}
func getTsDestPath(dir string) string {
	return dir + "video-%05d.ts"
}
func MP4TransToM3U8(path string, dir string, rateStr string) (err error) {
	rate := getVideoRate(rateStr)
	if rate == "" {
		err = errors.New(fmt.Sprintf("failed to get video rate:[%s]", rateStr))
		return
	}
	cmd := exec.Command("ffmpeg", "-y", "-i", path, "-vcodec", "h264", "-crf", "30","-g", rate, "-acodec", "libfdk_aac", "-movflags", "+faststart", "-strict", "experimental", "-f", "segment", "-segment_time", "10", "-segment_list", getM3u8DestPath(dir), "-sc_threshold", "0", getTsDestPath(dir))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		writeLogError(fmt.Sprintf("MP4TransToM3U8 failed, error:[%s]", err.Error()))
		return
	}
	return
}

type GetReq struct {
	Name string `json:"name"`
}
type GetResp struct {
	Endpoint         string `json:"endpoint"`
	EndpointInternal string `json:"endpoint_internal"`
	AccessKey        string `json:"access_key"`
	SecretKey        string `json:"secret_key"`
	SecurityToken    string `json:"security_token"`
	RequestID        string `json:"request_id"`
	ExpireSec        int    `json:"expire_sec"`
	Bucket           string `json:"bucket"`
}
type GetRespData struct {
	Ret     int   		`json:"ret"`
	Data    GetResp		`json:"data"`
}

func getTokenReqByEnv(env string) string {
	if env == EnvProd {
		return "https://kapi.xiaoniangao.cn/resources_center/uptoken/get"
	} else {
		return "http://192.168.11.50:8987/uptoken/get"
	}
}
func GetUploadToken(name string, env string) (response *GetRespData, err error) {
	url := getTokenReqByEnv(env)
	request := GetReq{Name: name}
	response = &GetRespData{}
	bts, _ := json.Marshal(request)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bts))
	if err != nil {
		return
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	if err != nil {
		return
	}
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

func getDirFileCount(dir string) (cnt int, err error) {
	files,err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}
	cnt = len(files)
	return
}
func UploadM3U8Video(dir string, key string, env string) (err error) {
	config, err := GetUploadToken("album", env)
	if err != nil {
		writeLogError(fmt.Sprintf("GetUploadToken failed, err:[%v]", err.Error()))
		return
	}
	writeLogInfo(fmt.Sprintf("GetUploadToken config:[%v]", config))
	//todo::判断环境变量使用endpoint
	var endpoint string
	//if env == EnvProd {
	//	endpoint = config.Data.EndpointInternal
	//} else {
	//	endpoint = config.Data.Endpoint
	//}
	endpoint = config.Data.Endpoint
	client, err := oss.New(endpoint, config.Data.AccessKey, config.Data.SecretKey, oss.SecurityToken(config.Data.SecurityToken))
	if err != nil {
		return
	}
	bucket, err := client.Bucket(config.Data.Bucket)
		if err != nil {
		return
	}
	m3u8Path := getM3u8DestPath(dir)
	objectKey := key + "/index.m3u8"
	err = bucket.PutObjectFromFile(objectKey, m3u8Path)
	if err != nil {
		writeLogError(fmt.Sprintf("PutObjectFromFile failed, err:[%v]", err.Error()))
		return
	}
	//todo::补充上传制作中预览的m3u8索引
	objectKey = key + "/index_0.m3u8"
	err = bucket.PutObjectFromFile(objectKey, m3u8Path)
	if err != nil {
		writeLogError(fmt.Sprintf("PutObjectFromFile failed, err:[%v]", err.Error()))
		return
	}
	count, err := getDirFileCount(dir)
	if err != nil {
		writeLogError(fmt.Sprintf("getDirFileCount failed, err:[%v]", err.Error()))
		return
	}
	tsCount := count - 2
	for i := 0; i < tsCount; i++ {
		baseName := fmt.Sprintf("video-%05d.ts", i)
		tsPath := dir + baseName
		tsName := key + "/" + baseName
		err = bucket.PutObjectFromFile(tsName, tsPath)
		if err != nil {
			writeLogError(fmt.Sprintf("PutObjectFromFile %s failed, err:[%v]",tsName, err.Error()))
			return
		}
	}
	return
}
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
func DealOneVideo(key string, env string) (err error) {
	//todo::下载资源
	url, err := GetVideoUrlByID(key, env)
	if err != nil {
		writeLogError(fmt.Sprintf("GetVideoUrlByID failed, err:[%v]", err.Error()))
		return
	}
	dir := fmt.Sprintf("/data/server/xngo/trans-m3u8/%s/", key)
	exists, _ := PathExists(dir)
	if !exists {
		err = os.Mkdir(dir, 0777)
		if err != nil {
			writeLogError(fmt.Sprintf("os.Mkdir failed, err:[%v]", err.Error()))
			return
		}
	}
	path, err := DownLoadResource(url, dir, key)
	if err != nil {
		writeLogError(fmt.Sprintf("DownLoadResource failed, err:[%v]", err.Error()))
		return
	}
	//todo::获取资源码率
	mediaInfo, err := GetMediaInfoByFFMPEG(path)
	if err != nil {
		writeLogError(fmt.Sprintf("GetMediaInfoByFFMPEG failed, err:[%v]", err.Error()))
		return
	}
	//todo::生成m3u8
	err = MP4TransToM3U8(path, dir, mediaInfo.Rate)
	if err != nil {
		writeLogError(fmt.Sprintf("MP4TransToM3U8 failed, err:[%v]", err.Error()))
		return
	}
	//todo::获取token，上传资源
	err = UploadM3U8Video(dir, key, env)
	return
}
func GetResIDsFromTXT(path string) (ids []string, err error) {
	bts, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	ids = strings.Split(string(bts), "\n")
	return
}

func writeLogError(msg string) {
	file, err := os.OpenFile(LogPathError, os.O_WRONLY | os.O_APPEND, 0666)
	if err != nil {
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	msgStr := fmt.Sprintf("%v: %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
	_, _ = writer.WriteString(msgStr)
	_ = writer.Flush()
}
func writeLogInfo(msg string) {
	file, err := os.OpenFile(LogPathInfo, os.O_WRONLY | os.O_APPEND, 0666)
	if err != nil {
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	msgStr := fmt.Sprintf("%v: %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
	_, _ = writer.WriteString(msgStr)
	_ = writer.Flush()
}
func AddFailIDs(id string) {
	file, err := os.OpenFile(FailedIds, os.O_WRONLY | os.O_APPEND, 0666)
	if err != nil {
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	_, _ = writer.WriteString(id + "\n")
	_ = writer.Flush()
}
func AddSuccessIDs(id string) {
	file, err := os.OpenFile(OkIds, os.O_WRONLY | os.O_APPEND, 0666)
	if err != nil {
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	_, _ = writer.WriteString(id + "\n")
	_ = writer.Flush()
}
func RemoveDir(dir string) (err error) {
	path := "./" + dir
	err = os.RemoveAll(path)
	return
}

func getExistsM3u8URL(env string) string {
	if env == EnvProd {
		return "https://kapi.xiaoniangao.cn/resources_center/resource/res_exists"
	} else {
		return "http://192.168.11.50:8987/resource/res_exists"
	}
}
type ReqCheckResExists struct {
	Kind int    `json:"type"`
	Qid  string `json:"id"`
}
type RespCheckResExists struct {
	IsExist bool `json:"is_exist"`
}
type RespCheckResExistsData struct {
	Ret     int   `json:"ret"`
	Data    RespCheckResExists  `json:"data"`
}
func ExistsM3u8(id string, env string) (exists bool, err error) {
	reqUrl := getExistsM3u8URL(env)
	request := ReqCheckResExists{Kind: 12, Qid:fmt.Sprintf("%s/index.m3u8",id)}
	response := &RespCheckResExistsData{}
	bts, _ := json.Marshal(request)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewReader(bts))
	if err != nil {
		return
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	if err != nil {
		return
	}
	exists = response.Data.IsExist
	return
}

func getAddM3u8ExpID(env string) string {
	if env == EnvProd {
		return "https://kapi.xiaoniangao.cn/resources_center/resource/set_m3u8_id"
	} else {
		return "http://192.168.11.50:8987/resource/set_m3u8_id"
	}
}
type ReqSetM3u8ExpID struct {
	IDs []string `json:"ids"`
}
type RespSetM3u8ExpID struct {
	Ret   int   `json:"ret"`
}
func AddM3u8ExpID(id string, env string) (err error) {
	reqUrl := getAddM3u8ExpID(env)
	ids := []string{id}
	request := ReqSetM3u8ExpID{IDs:ids}
	response := &RespSetM3u8ExpID{}
	bts, _ := json.Marshal(request)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewReader(bts))
	if err != nil {
		return
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	if err != nil {
		return
	}
	if response.Ret != 1 {
		err = errors.New(fmt.Sprintf("add exp id failed, ret code:[%v]", response.Ret))
	}
	return
}

func main() {
	if len(os.Args) < 2 {
		return
	}
	var env string
	env = os.Args[1]
	writeLogInfo(fmt.Sprintf("start with env:[%s]", env))
	ids, err := GetResIDsFromTXT(ResIDsFile)
	if err != nil {
		return
	}
	for _, id := range ids {
		if id == "" {
			continue
		}
		exists, errIgnore:= ExistsM3u8(id, env)
		if errIgnore != nil {
			writeLogError(errIgnore.Error())
			AddFailIDs(id)
			continue
		}
		if exists == true {
			AddSuccessIDs(id)
			writeLogInfo(fmt.Sprintf("id:%s already exists", id))
			errIgnore = AddM3u8ExpID(id, env)
			if errIgnore != nil {
				writeLogError(errIgnore.Error())
				continue
			}
			continue
		}
		errIgnore = DealOneVideo(id, env)
		if errIgnore != nil {
			writeLogError(errIgnore.Error())
			AddFailIDs(id)
			continue
		}
		AddSuccessIDs(id)
		errIgnore = AddM3u8ExpID(id, env)
		if errIgnore != nil {
			writeLogError(errIgnore.Error())
			AddFailIDs(id)
			continue
		}
		if env == EnvProd {
			_ =  RemoveDir(id)
		}
	}
}
