package res_url

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/conf"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/util/common"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/util/net"
)

const (
	ServiceName         = "resources-center"
	ServiceVideoPath    = "url/video"
	ServiceImgPath      = "url/img"
	ServiceTestVideoUrL = "http://127.0.0.1:8987/url/video"
	ServiceTestImgUrL   = "http://127.0.0.1:8987/url/img"
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

func GetVideoUrlByID(ctx context.Context, key string) (url string, err error) {
	var request []ReqVideoUrl
	param := ReqVideoUrl{ID: key}
	request = append(request, param)
	response := RespVideoUrl{}
	if conf.Env == lib.PROD {
		err = net.XngServiceCallPostWithRetry(ctx, &net.Consul{}, ServiceName, ServiceVideoPath, request, &response, time.Second, 1)
		if err != nil {
			return
		}
	} else {
		bts, _ := json.Marshal(request)
		req, errIngro := http.NewRequest("POST", ServiceTestVideoUrL, bytes.NewReader(bts))
		if errIngro != nil {
			err = errIngro
			return
		}
		client := http.Client{}
		resp, errIngro := client.Do(req)
		if errIngro != nil {
			err = errIngro
			return
		}
		defer func() {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()
		body, _ := ioutil.ReadAll(resp.Body)
		xlog.DebugC(ctx, "get url req:[%v], resp body:[%s]", req, string(body))
		errIngro = json.Unmarshal(body, &response)
		if errIngro != nil {
			err = errIngro
			return
		}
	}
	if conf.Env == lib.PROD {
		url = response.Data.URLs[key].URLInternal
	} else {
		url = response.Data.URLs[key].URL
	}
	xlog.DebugC(ctx, "get url response:[%v], url:[%s]", response, url)
	return
}

type ReqImgUrl struct {
	ID string `json:"id" binding:"required"`
	QS string `json:"qs" binding:"required"`
}
type RespImgUrl RespVideoUrl

func GetImgUrlByID(ctx context.Context, key string) (url string, urlInternal string, err error) {
	var request []ReqImgUrl
	param := ReqImgUrl{ID: key}
	request = append(request, param)
	response := RespImgUrl{}
	if conf.Env == lib.PROD {
		err = net.XngServiceCallPostWithRetry(ctx, &net.Consul{}, ServiceName, ServiceImgPath, request, &response, time.Second, 1)
		if err != nil {
			return
		}
	} else {
		bts, _ := json.Marshal(request)
		req, errIngro := http.NewRequest("POST", ServiceTestImgUrL, bytes.NewReader(bts))
		if errIngro != nil {
			err = errIngro
			return
		}
		client := http.Client{}
		resp, errIngro := client.Do(req)
		if errIngro != nil {
			err = errIngro
			return
		}
		defer func() {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()
		body, _ := ioutil.ReadAll(resp.Body)
		errIngro = json.Unmarshal(body, &response)
		if errIngro != nil {
			err = errIngro
			return
		}
	}
	url = response.Data.URLs[key].URL
	urlInternal = response.Data.URLs[key].URLInternal
	return
}
func DownLoadResource(ctx context.Context, url string, key string) (path string, err error) {
	response, err := http.Get(url)
	if err != nil {
		return
	}
	path = common.GetSrcPath(key)
	//path = "./src_video/" + key + ".mp4"
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
