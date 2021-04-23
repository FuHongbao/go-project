package url

import (
	"context"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils/alioss"
)

// GetAlbumURL 获取影集播放地址
func GetAlbumURL(ctx context.Context, key string, process int) (url string, albumInternalURL string) {
	signURL := alioss.GetAlbumSignURL(ctx, key, process)
	aliDomain := getAlbumCDNDomain(ctx, TypeAliOSS)
	url = alioss.SignedURL2CdnURL(ctx, signURL, aliDomain)
	albumInternalDomain := getInternalDomain(ctx, TypeInternalAlbumAli)
	albumInternalURL = alioss.SignedURL2CdnURL(ctx, signURL, albumInternalDomain)
	return
}
