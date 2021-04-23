package url

import (
	"context"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils/alioss"
)

// GetLyricURL 获取歌词资源地址
func GetLyricURL(ctx context.Context, key string) (url string, lyricInternalURL string) {
	signURL := alioss.GetImageSignURL(ctx, key, nil)
	domain := getCDNDomain(ctx, "img_ali")
	url = alioss.SignedURL2CdnURL(ctx, signURL, domain)
	lyricInternalDomain := getInternalDomain(ctx, TypeInternalResourceAli)
	lyricInternalURL = alioss.SignedURL2CdnURL(ctx, signURL, lyricInternalDomain)
	return url, lyricInternalURL
}
