package speech

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	SpeechDao "xgit.xiaoniangao.cn/xngo/service/resources_center/dao/speech"
)

const (
	NLSTokenVersion   = "2019-02-28"
	NLSTokenApiName   = "CreateToken"
	NLSTokenLifeLimit = 600
)

type NLSTokenData struct {
	UserId     string `json:"UserId"`
	Id         string `json:"Id"`
	ExpireTime int    `json:"ExpireTime"`
}
type NLSSpeechToken struct {
	ErrMsg string       `json:"ErrMsg"`
	Token  NLSTokenData `json:"Token"`
}

func GetSpeechTokenFromNLS(ctx context.Context) (token string, expireTime int, err error) {
	client, err := sdk.NewClientWithAccessKey(conf.C.Speech.Region, conf.C.Speech.AccessKeyId, conf.C.Speech.AccessKeySecret)
	if err != nil {
		return
	}
	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.Domain = conf.C.Speech.Domain
	request.ApiName = NLSTokenApiName
	request.Version = NLSTokenVersion
	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		return
	}
	respData := NLSSpeechToken{}
	err = json.Unmarshal(response.GetHttpContentBytes(), &respData)
	if err != nil {
		return
	}
	token = respData.Token.Id
	expireTime = respData.Token.ExpireTime
	xlog.DebugC(ctx, "GetSpeechToken http status:[%s], content:[%s]", response.GetHttpStatus(), response.GetHttpContentString())
	return
}

func GetSpeechToken(ctx context.Context) (token string, err error) {
	token, ext, err := SpeechDao.GetSpeechToken(ctx)
	if err != nil {
		return
	}
	if token != "" {
		xlog.DebugC(ctx, "token life ext:[%d]", ext)
		return
	}
	token, _, err = GetSpeechTokenFromNLS(ctx)
	if err != nil {
		return
	}
	if token == "" {
		err = errors.New("failed to get speech token")
		return
	}
	err = SpeechDao.SetSpeechToken(ctx, token, NLSTokenLifeLimit)
	return
}
