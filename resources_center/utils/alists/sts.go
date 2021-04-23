package alists

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"github.com/satori/go.uuid"
	"net/url"
	"strings"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils/net"
)

const (
	//detail information:https://help.aliyun.com/document_detail/66053.html?spm=a2c4g.11186623.2.12.151338dfkKJJKh
	// StsSignVersion sts sign version
	StsSignVersion = "1.0"
	// StsAPIVersion sts api version
	StsAPIVersion = "2015-04-01"
	// TimeFormat time fomrat
	TimeFormat = "2006-01-02T15:04:05Z"
	// RespBodyFormat  respone body format
	RespBodyFormat = "JSON"
	// PercentEncode '/'
	PercentEncode = "%2F"
	// HTTPGet http get method
	HTTPGet = "GET"
)

//Credentials for token server get success
type Credentials struct {
	AccessKeyId     string `json:"AccessKeyId"`
	AccessKeySecret string `json:"AccessKeySecret"`
	Expiration      string `json:"Expiration"`
	SecurityToken   string `json:"SecurityToken"`
}

//AssumedRoleUser for token server get success
type AssumedRoleUser struct {
	Arn           string `json:"Arn"`
	AssumedRoleId string `json:"AssumedRoleId"`
}

//StsResponse the response of sts service
type StsResponse struct {
	Credentials     Credentials     `json:"Credentials"`
	AssumedRoleUser AssumedRoleUser `json:"AssumedRoleUser"`
	RequestId       string          `json:"RequestId"`
}

// token server config
type StsConfig struct {
	AccessKeyId      string `mapstructure:"access_key_id"`
	AccessKeySecret  string `mapstructure:"access_key_secret"`
	RoleArn          string `mapstructure:"role_arn"`
	TokenExpireTime  int    `mapstructure:"token_expire_time"`
	Bucket           string `mapstructure:"bucket"`
	Endpoint         string `mapstructure:"endpoint"`
	EndpointInternal string `mapstructure:"endpoint_internal"`
	SessionName      string `mapstructure:"session_name"`
	ReqHost          string `mapstructure:"req_host"`
}

// generateSignedURL Private function
func generateSignedURL(stsConfig *StsConfig, policy string) (string, error) {
	id := uuid.NewV4()

	queryStr := fmt.Sprintf("SignatureVersion=%s&Format=%s&Timestamp=%s&RoleArn=%s&RoleSessionName=%s&AccessKeyId=%s&SignatureMethod=HMAC-SHA1&Version=%s&Action=AssumeRole&SignatureNonce=%s&DurationSeconds=%d", StsSignVersion, RespBodyFormat, url.QueryEscape(time.Now().UTC().Format(TimeFormat)), url.QueryEscape(stsConfig.RoleArn), stsConfig.SessionName, stsConfig.AccessKeyId, StsAPIVersion, id.String(), stsConfig.TokenExpireTime)

	if policy != "" {
		queryStr += "&Policy=" + url.QueryEscape(policy)
	}

	// Sort query string
	queryParams, err := url.ParseQuery(queryStr)
	if err != nil {
		return "", err
	}
	sortUrl := strings.Replace(queryParams.Encode(), "+", "%20", -1)
	strToSign := HTTPGet + "&" + PercentEncode + "&" + url.QueryEscape(sortUrl)

	// Generate signature
	hashSign := hmac.New(sha1.New, []byte(stsConfig.AccessKeySecret+"&"))
	hashSign.Write([]byte(strToSign))
	signature := base64.StdEncoding.EncodeToString(hashSign.Sum(nil))

	// Build url
	assumeURL := stsConfig.ReqHost + "?" + queryStr + "&Signature=" + url.QueryEscape(signature)
	return assumeURL, nil
}

func GetSts(ctx context.Context, stsConfig *StsConfig, policy string, isDev bool) (resp *StsResponse, err error) {
	signURL, err := generateSignedURL(stsConfig, policy)
	if err != nil {
		return
	}

	resp = &StsResponse{}
	timeOut := time.Second
	if isDev == true {
		timeOut = time.Minute
	}
	err = net.Get(ctx, signURL, timeOut, resp)
	if err != nil {
		xlog.ErrorC(ctx, "fail to get sts, url:%s, resp:%v, err:%v", signURL, resp, err)
		return
	}
	return
}
