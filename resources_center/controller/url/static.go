package url

import (
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gin-gonic/gin"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/sts"
	urlService "xgit.xiaoniangao.cn/xngo/service/resources_center/service/url"
)

// StaticReq ..
type StaticReq struct {
	FileName string `json:"filename"`
	YWSide   string `json:"yw_side"`
	Path     string `json:"path"`
	Prod     int    `json:"prod"`
}

// StaticReq ...
type StaticResp struct {
	Url         string `json:"url"`
	UrlInternal string `json:"url_internal"`
}

func StaticURL(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req StaticReq
	if !xc.GetReqObject(&req) {
		return
	}
	stsData, err := sts.GetUpToken(xc, api.StsForStaticUpload)
	if err != nil {
		xlog.ErrorC(xc, "failed to GetUpToken, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if stsData == nil {
		xlog.ErrorC(xc, "failed to GetUpToken, req:%v, data is nil", req)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	path := ""
	if req.Path != "" {
		path = req.Path + "/"
	}
	ObjectKey := api.StaticUploadMap[req.Prod] + "/" + req.YWSide + "/" + path + req.FileName
	url, urlInternal := urlService.GetStaticURL(xc, ObjectKey)
	resp := StaticResp{
		//Url:         fmt.Sprintf("https://%s.%s/%s", stsData.Bucket, stsData.Endpoint, ObjectKey),
		//UrlInternal: fmt.Sprintf("https://%s.%s/%s", stsData.Bucket, stsData.EndpointInternal, ObjectKey),
		Url:         url,
		UrlInternal: urlInternal,
	}
	xc.ReplyOK(resp)
}

type StaticImgReq struct {
	Filename string   `json:"filename" binding:"required"`
	QS       []string `json:"qs" binding:"required"`
	Ty       int      `json:"ty"`
}

type StaticUrlData struct {
	URLs []string `json:"urls"`
}

func StaticImgURL(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req []StaticImgReq
	if !xc.GetReqObject(&req) {
		return
	}
	stsData, err := sts.GetUpToken(xc, api.StsForStaticUpload)
	if err != nil {
		xlog.ErrorC(xc, "failed to GetUpToken, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if stsData == nil {
		xlog.ErrorC(xc, "failed to GetUpToken, req:%v, data is nil", req)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	var endpoint string
	if conf.Env == lib.PROD {
		endpoint = stsData.EndpointInternal
	} else {
		endpoint = stsData.Endpoint
	}
	client, err := oss.New(endpoint, stsData.AccessKey, stsData.SecretKey, oss.SecurityToken(stsData.SecurityToken))
	if err != nil {
		xlog.ErrorC(xc, "failed to create oss client, req:%v, err:%v", req, err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	bucket, err := client.Bucket(stsData.Bucket)
	resp := make(map[string]StaticUrlData)
	for _, v := range req {
		if v.Ty != api.ResourceTypeImg {
			continue
		}
		filename := v.Filename
		var urls []string
		for _, qs := range v.QS {
			url, err := urlService.GetStaticImageURL(xc, filename, qs, bucket)
			if err != nil {
				xlog.ErrorC(xc, "failed to GetStaticImageURL, err:%v", err)
			}
			urls = append(urls, url)
		}
		respUrl := StaticUrlData{URLs: urls}
		resp[filename] = respUrl
	}
	xc.ReplyOK(resp)
}
