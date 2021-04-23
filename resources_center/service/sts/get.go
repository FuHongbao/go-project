package sts

import (
	"context"
	"fmt"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	stsDao "xgit.xiaoniangao.cn/xngo/service/resources_center/dao/sts"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils/alists"
)

const stsCacheTime = 300

// GetStsConfig 获取config
func GetStsConfig(ctx context.Context, name string) (config *alists.StsConfig, err error) {
	if c, ok := conf.C.Sts[name]; ok {
		config = c
		return
	}
	return nil, fmt.Errorf("fail to get [%s] sts config", name)
}

type UploadToken struct {
	Endpoint         string `json:"endpoint"`
	EndpointInternal string `json:"endpoint_internal"`
	AccessKey        string `json:"access_key"`
	SecretKey        string `json:"secret_key"`
	SecurityToken    string `json:"security_token"`
	RequestID        string `json:"request_id"`
	ExpireSec        int    `json:"expire_sec"`
	Bucket           string `json:"bucket"`
}

// GetUpToken 获取token
func GetUpToken(ctx context.Context, name string) (upToken *UploadToken, err error) {
	c, err := GetStsConfig(ctx, name)
	if err != nil {
		return
	}

	resp := &alists.StsResponse{}
	var ext int

	r, extR, err := stsDao.GetUploadToken(ctx, name)
	if err != nil {
		xlog.ErrorC(ctx, "get uptoken from cache err:%v, name:%s", err, name)
		err = nil
	}

	if r != nil {
		resp = r
		ext = extR
	} else {
		var dev bool
		if conf.Env != lib.PROD {
			dev = true
		} else {
			dev = false
		}
		rAli, err1 := alists.GetSts(ctx, c, "", dev)
		if err1 != nil {
			err = err1
			return
		}
		resp = rAli
		ext = stsCacheTime
		_ = stsDao.SetUploadToken(ctx, name, rAli, ext)
	}

	upToken = &UploadToken{
		Endpoint:         c.Endpoint,
		EndpointInternal: c.EndpointInternal,
		AccessKey:        resp.Credentials.AccessKeyId,
		SecretKey:        resp.Credentials.AccessKeySecret,
		SecurityToken:    resp.Credentials.SecurityToken,
		RequestID:        resp.RequestId,
		ExpireSec:        ext,
		Bucket:           c.Bucket,
	}
	return
}
