package auth

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/auth"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/resource"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/sts"
)

func GetMultiPartAuth(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req api.ReqMultiPartAuth
	if !xc.GetReqObject(&req) {
		return
	}
	if req.Key == "" || req.UploadID == "" || req.ChunkNum == 0 {
		xc.ReplyFail(lib.CodePara)
		return
	}
	stsName := resource.GetUploadStsName(req.Kind)
	stsData, err := sts.GetUpToken(xc, stsName)
	if err != nil {
		return
	}
	if stsData == nil {
		err = errors.New("uptoken data is nil")
		return
	}
	endpoint := stsData.Endpoint
	//if conf.Env == lib.PROD {
	//	endpoint = stsData.EndpointInternal
	//}
	host := stsData.Bucket + "." + endpoint
	var ObjectBuf strings.Builder
	ObjectBuf.WriteString("/")
	ObjectBuf.WriteString(req.Key)
	ObjectBuf.WriteString("?partNumber=")
	ObjectBuf.WriteString(strconv.Itoa(req.ChunkNum))
	ObjectBuf.WriteString("&uploadId=")
	ObjectBuf.WriteString(req.UploadID)
	var resp api.RespMultiPartAuth
	resp.Url = "https://" + host + ObjectBuf.String()
	Date := time.Now().UTC().Format(http.TimeFormat)
	ossHeaders := "x-oss-security-token:" + stsData.SecurityToken + "\n"
	ObjectPath := "/" + stsData.Bucket + ObjectBuf.String()
	signature, err := auth.GetHeaderSignature(http.MethodPut, stsData.SecretKey, req.Md5Value, api.MultiPartContentType, Date, ossHeaders, ObjectPath)
	if err != nil {
		xlog.ErrorC(xc, "failed to GetMultiPartAuth, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resp.Authorization = auth.GetAuthorization(stsData.AccessKey, signature)
	resp.Date = Date
	resp.Token = stsData.SecurityToken
	resp.Host = host
	resp.ExpireSec = stsData.ExpireSec
	resp.Method = http.MethodPut
	resp.ContentType = api.MultiPartContentType
	xc.ReplyOK(resp)
}
