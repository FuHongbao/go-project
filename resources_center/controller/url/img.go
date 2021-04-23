package url

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	resourceService "xgit.xiaoniangao.cn/xngo/service/resources_center/service/resource"
	urlService "xgit.xiaoniangao.cn/xngo/service/resources_center/service/url"
)

// ImgReq ..
type ImgReq []struct {
	ID string `json:"id" binding:"required"`
	QS string `json:"qs" binding:"required"`
}

type ImgResp struct {
	URLs map[string]RespURL `json:"urls"`
}

func ImgURL(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req ImgReq
	if !xc.GetReqObject(&req) {
		return
	}
	total := time.Now()
	conf.UrlImgCounter.Inc()
	resp := &ImgResp{URLs: make(map[string]RespURL)}

	for _, v := range req {
		url, urlInternal := urlService.GetImageURL(xc, v.ID, v.QS)
		resp.URLs[v.ID] = RespURL{URL: url, URLInternal: urlInternal}
	}
	xlog.DebugC(xc, "ImgURL use time:[%v] total", time.Since(total))
	xc.ReplyOK(resp)
}

type ImgReqOne struct {
	ID         string             `json:"id" binding:"required"`
	QS         []string           `json:"qs" binding:"required"`
	Ty         int                `json:"ty"`
	WithNoMark bool               `json:"with_no_mark"`
	Watermark  []api.WatermarkOne `json:"watermark"`
}

type ImgRespV2 map[string]RespURLs

func ImgURLV2(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req []ImgReqOne
	if !xc.GetReqObject(&req) {
		return
	}
	conf.UrlImgV2Counter.Inc()
	resp := make(map[string]RespURLs)
	start := time.Now()
	vIDm := map[string]ImgReqOne{}
	var vIDs []string
	for _, v := range req {
		var urls, urlsNoMark []string
		var urlInternals, urlNoMarkInternals []string
		respUrls := RespURLs{}

		imgID := v.ID

		if v.Ty == api.ResourceTypeVideo {
			vIDm[v.ID] = v
			vIDs = append(vIDs, v.ID)
		} else {
			for _, qs := range v.QS {
				url, urlInternal := urlService.GetImageURLWithWatermark(xc, imgID, qs, v.Watermark)
				urls = append(urls, url)
				urlInternals = append(urlInternals, urlInternal)
			}
			if v.WithNoMark == true {
				for _, qs := range v.QS {
					url, urlInternal := urlService.GetImageURL(xc, imgID, qs)
					urlsNoMark = append(urlsNoMark, url)
					urlNoMarkInternals = append(urlNoMarkInternals, urlInternal)
				}
			}
			respUrls.URLs = urls
			respUrls.URLInternals = urlInternals
			respUrls.URLsNoMark = urlsNoMark
			respUrls.URLsNoMarkInternal = urlNoMarkInternals
			resp[v.ID] = respUrls
		}
	}
	if len(vIDs) > 0 {
		qDocs, err := resourceService.ByIDs(xc, vIDs)
		if err != nil {
			//xlog.ErrorC(xc, "fail to get docs, vids:%v, err:%v", vIDs, err)
			xc.ReplyFail(lib.CodeSrv)
			return
		}

		for id, v := range qDocs {
			vUrl, vUrlInternal := urlService.GetVideoURL(xc, id)
			imgID := strconv.FormatInt(v.Cover, 10)
			var urls []string
			var urlInternals []string
			for _, qs := range vIDm[id].QS {
				url, urlInternal := urlService.GetImageURL(xc, imgID, qs)
				xlog.DebugC(xc, "url:%s", url)
				urls = append(urls, url)
				urlInternals = append(urlInternals, urlInternal)
			}
			resp[id] = RespURLs{
				URLs:             urls,
				VideoUrl:         vUrl,
				URLInternals:     urlInternals, //内网截图url
				VideoUrlInternal: vUrlInternal, //内网视频url
			}
		}
	}
	xlog.DebugC(xc, "ImgURLV2 use time total :[%d]", time.Since(start))
	xc.ReplyOK(resp)
}
