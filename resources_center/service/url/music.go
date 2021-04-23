package url

import (
	"context"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils/alioss"
)

// GetAuditMusicURL 获取审核音频地址
func GetAuditMusicURL(ctx context.Context, key string) (url string, videoInternalURL string) {
	signURL := alioss.GetAuditMusicSignURL(ctx, key)
	domain := getCDNDomain(ctx, "audit_music")
	url = alioss.SignedURL2CdnURL(ctx, signURL, domain)
	videoInternalDomain := getInternalDomain(ctx, TypeInternalAuditMusic)
	videoInternalURL = alioss.SignedURL2CdnURL(ctx, signURL, videoInternalDomain)
	return url, videoInternalURL
}

func GetMusicURL(ctx context.Context, key string) (url string, videoInternalURL string) {
	signURL := alioss.GetImageSignURL(ctx, key, nil)
	domain := getCDNDomain(ctx, "img_ali")
	url = alioss.SignedURL2CdnURL(ctx, signURL, domain)
	videoInternalDomain := getInternalDomain(ctx, TypeInternalResourceAli)
	videoInternalURL = alioss.SignedURL2CdnURL(ctx, signURL, videoInternalDomain)
	return
}
