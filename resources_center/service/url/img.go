package url

import (
	"context"
	"encoding/base64"
	"strings"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils/alioss"
)

// GetImageURL 获取图片地址
func GetImageURL(ctx context.Context, key, qs string) (imageURL string, imgInternalURL string) {
	actions := alioss.ParseQs(qs)
	signURL := alioss.GetImageSignURL(ctx, key, actions)
	imgDomain := getCDNDomain(ctx, "img_ali")
	imageURL = alioss.SignedURL2CdnURL(ctx, signURL, imgDomain)
	imgInternalDomain := getInternalDomain(ctx, TypeInternalResourceAli)
	imgInternalURL = alioss.SignedURL2CdnURL(ctx, signURL, imgInternalDomain)
	return imageURL, imgInternalURL
}

// GetImageURLWithWatermark 获取水印图片地址
func GetImageURLWithWatermark(ctx context.Context, key, qs string, watermarks []api.WatermarkOne) (imageURL string, imgInternalURL string) {
	actions := alioss.ParseQs(qs)
	//start := time.Now()
	//水印
	for _, wm := range watermarks {
		if wm.Ty == api.TypeWatermarkImg {
			imgData := alioss.GetWatermarkData(ctx, wm.Key, alioss.ParseQs(wm.QS))
			wmAction := alioss.ParseWatermarkQs(wm.WatermarkQS)
			wmAction.Image = imgData
			actions = append(actions, wmAction)
		} else if wm.Ty == api.TypeWatermarkText {
			wmAction := alioss.ParseWatermarkQs(wm.WatermarkQS)
			t := base64.StdEncoding.EncodeToString([]byte(wm.Key))
			t = strings.ReplaceAll(t, "+", "-")
			t = strings.ReplaceAll(t, "/", "_")
			wmAction.Text = t
			actions = append(actions, wmAction)
		}
	}
	//xlog.DebugC(ctx, "GetImageURLWithWatermark.ParseWatermarkQs use time :[%d]", time.Since(start))
	//start = time.Now()
	signURL := alioss.GetImageSignURL(ctx, key, actions)
	//xlog.DebugC(ctx, "GetImageURLWithWatermark.GetImageSignURL use time :[%d]", time.Since(start))
	//start = time.Now()
	imgDomain := getCDNDomain(ctx, "img_ali")
	imageURL = alioss.SignedURL2CdnURL(ctx, signURL, imgDomain)
	//xlog.DebugC(ctx, "GetImageURLWithWatermark.SignedURL2CdnURL use time :[%d]", time.Since(start))
	//start = time.Now()
	imgInternalDomain := getInternalDomain(ctx, TypeInternalResourceAli)
	imgInternalURL = alioss.SignedURL2CdnURL(ctx, signURL, imgInternalDomain)
	//xlog.DebugC(ctx, "GetImageURLWithWatermark.SignedURL2CdnURL use time :[%d]", time.Since(start))
	//start = time.Now()
	return imageURL, imgInternalURL
}
