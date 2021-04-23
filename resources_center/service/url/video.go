package url

import (
	"context"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils/alioss"
)

// GetVideoURL 获取视频地址
func GetVideoURL(ctx context.Context, key string) (url string, videoInternalURL string) {
	signURL := alioss.GetImageSignURL(ctx, key, nil)
	domain := getCDNDomain(ctx, "img_ali")
	url = alioss.SignedURL2CdnURL(ctx, signURL, domain)
	videoInternalDomain := getInternalDomain(ctx, TypeInternalResourceAli)
	videoInternalURL = alioss.SignedURL2CdnURL(ctx, signURL, videoInternalDomain)
	return url, videoInternalURL
}
