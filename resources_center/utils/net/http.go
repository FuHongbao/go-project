package net

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xconsul/xagent"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/lib/xnet/xhttp"
)

// 超时时间
const (
	NetTimeout       = time.Duration(200 * time.Millisecond)
	NetMiddleTimeout = time.Millisecond * 400
	NetLongTimeout   = time.Second * 2
	NetTestTimeout   = time.Second * 5
)

// ServiceDiscovery interface
type ServiceDiscovery interface {
	GetIPPortByNameSrv(ctx context.Context, nameStr string) (ipPort string, err error)
}

// GetURLByNameSrv get url ip port from name str
func GetURLByNameSrv(ctx context.Context, sd ServiceDiscovery, nameStr string) (ipPort string, err error) {
	if nameStr == "" {
		return "", errors.New("name str addr err")
	}
	ipPort, err = sd.GetIPPortByNameSrv(ctx, nameStr)
	if err != nil {
		return
	}
	return
}

// NameSrv struct
type NameSrv struct {
}

// GetIPPortByNameSrv namesrv
func (ns *NameSrv) GetIPPortByNameSrv(ctx context.Context, nameStr string) (ipPort string, err error) {
	ipPort, err = lib.NameWrap(nameStr)
	if err != nil {
		xlog.ErrorC(ctx, "fail to get ipPort by nameSrv, name str:%s, err:%v", nameStr, err)
		return "", err
	}
	return
}

// Consul name srv struct
type Consul struct {
}

// GetIPPortByNameSrv consul name srv
func (c *Consul) GetIPPortByNameSrv(ctx context.Context, nameStr string) (ipPort string, err error) {
	// todo implementation
	//不是validService，默认为ip或者域名，如果写ip的话，请在ip后面带上端口
	if !xagent.IsValidServiceName(nameStr) {
		ipPort = nameStr
	} else {
		ipPort = "127.0.0.1:9090/" + nameStr
	}
	return
}

// XngServiceCallGet xng service get call
func XngServiceCallGet(ctx context.Context, sd ServiceDiscovery, nameStr, path string, resp interface{}, timeout time.Duration) (err error) {
	ipPort, err := GetURLByNameSrv(ctx, sd, nameStr)
	if err != nil {
		xlog.ErrorC(ctx, "fail get ipPort by nameSrv, name str:%s, err:%v", nameStr, err)
		return
	}
	url := fmt.Sprintf("http://%s/%s", ipPort, path)
	err = Get(ctx, url, timeout, resp)
	if err != nil {
		xlog.ErrorC(ctx, "call service get err:%v, url:%s, resp:%v", err, url, resp)
		return
	}
	return
}

// Post post request
func Post(ctx context.Context, url string, timeOut time.Duration, param interface{}, resp interface{}) error {
	if timeOut == 0 {
		timeOut = NetTimeout
	}
	c := xhttp.NewClient()
	c.BackOff = xhttp.Linear
	c.Timeout = timeOut
	req, err := GetReq(ctx, http.MethodPost, url, param)
	if err != nil {
		xlog.ErrorC(ctx, "post req err:%v, params:%v, url:%s", err, param, url)
		return err
	}
	err = Call(ctx, c, req, resp)
	if err != nil {
		xlog.ErrorC(ctx, "post call err:%v, params:%v, url:%s", err, param, url)
		return err
	}
	xlog.DebugC(ctx, "post call, url:%s, params:%v, req:%v, resp:%v", url, param, req, resp)
	return nil
}

// Get get request
func Get(ctx context.Context, url string, timeout time.Duration, resp interface{}) error {
	if timeout == 0 {
		timeout = NetTimeout
	}
	c := xhttp.NewClient()
	c.BackOff = xhttp.Linear
	c.Timeout = timeout
	req, err := GetReq(ctx, http.MethodGet, url, nil)
	if err != nil {
		xlog.ErrorC(ctx, "get req err:%v, url:%s", err, url)
		return err
	}
	err = Call(ctx, c, req, resp)
	if err != nil {
		xlog.ErrorC(ctx, "post call err:%v, url:%s", err, url)
		return err
	}
	xlog.DebugC(ctx, "get call, url:%s, req, %v, resp:%v", url, req, resp)
	return nil
}

// GetReq get http request
func GetReq(ctx context.Context, method, url string, params interface{}) (req *http.Request, err error) {
	//traceID := GetTraceIDFromContext(ctx)
	switch method {
	case http.MethodGet:
		req, err = http.NewRequest(method, url, nil)
	case http.MethodPost:
		body, err1 := json.Marshal(params)
		if err1 != nil {
			return nil, err1
		}
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
	}
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	// 链路追踪信息
	err = xng.JaegerInjectToHeader(ctx, req.Header)
	if err != nil {
		return
	}
	xlog.DebugC(ctx, "req:%v, header:%v", req, req.Header)
	return req, nil
}

// Call is client Do
func Call(ctx context.Context, client *xhttp.Client, req *http.Request, result interface{}) (err error) {
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	// do not close resp.Body for outer reuse
	bodyResp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	xlog.DebugC(ctx, "body:%s", string(bodyResp))
	err = json.Unmarshal(bodyResp, &result)
	if err != nil {
		xlog.ErrorC(ctx, "unmarshal err:%v, body:%v", err, string(bodyResp))
		return
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status code:%d", resp.StatusCode)
	}
	return
}

// XngServiceCallPostWithRetry xng service post call with retry, 如果没有retry 把retry写为1
func XngServiceCallPostWithRetry(ctx context.Context, sd ServiceDiscovery, nameStr, path string, params interface{}, resp interface{}, timeout time.Duration, retry int) (err error) {
	ipPort, err := GetURLByNameSrv(ctx, sd, nameStr)
	if err != nil {
		xlog.ErrorC(ctx, "fail get ipPort by nameSrv, name str:%s, err:%v", nameStr, err)
		return
	}
	url := fmt.Sprintf("http://%s/%s", ipPort, path)
	err = PostWithRetry(ctx, url, timeout, params, resp, retry)
	if err != nil {
		xlog.ErrorC(ctx, "call service post err:%v, url:%s, params:%v, resp:%v", err, url, params, resp)
		return
	}
	return
}

// PostWithRetry post request
func PostWithRetry(ctx context.Context, url string, timeOut time.Duration, param interface{}, resp interface{}, retry int) error {
	if timeOut == 0 {
		timeOut = NetTimeout
	}
	c := xhttp.NewClient()
	c.BackOff = xhttp.Linear
	c.Timeout = timeOut
	c.Retries = retry
	req, err := GetReq(ctx, http.MethodPost, url, param)
	if err != nil {
		xlog.ErrorC(ctx, "post req err:%v, params:%v, url:%s", err, param, url)
		return err
	}
	err = Call(ctx, c, req, resp)
	if err != nil {
		xlog.ErrorC(ctx, "post call err:%v, params:%v, url:%s", err, param, url)
		return err
	}
	xlog.DebugC(ctx, "post call, url:%s, params:%v, req:%v, resp:%v", url, param, req, resp)
	return nil
}
