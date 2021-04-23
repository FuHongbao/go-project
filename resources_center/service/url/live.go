package url

import (
	"context"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils/alioss"
)

// GetAlbumURL 获取直播视频播放地址
func GetLiveVideoURL(ctx context.Context, key string) (url string, liveInternalURL string) {
	signURL := alioss.GetLiveVideoSignURL(ctx, key)
	aliDomain := getLiveCDNDomain(ctx, TypeAliOSS)
	url = alioss.SignedURL2CdnURL(ctx, signURL, aliDomain)
	liveInternalDomain := getInternalDomain(ctx, TypeInternalLiveAli)
	liveInternalURL = alioss.SignedURL2CdnURL(ctx, signURL, liveInternalDomain)
	return
}

//  获取直播导播视频播放地址
func GetLiveGuideVideoURL(ctx context.Context, key string) (url string, liveInternalURL string) {
	signURL := alioss.GetLiveGuideSignURL(ctx, key)
	aliDomain := getCDNDomain(ctx, "live_guide_ali")
	url = alioss.SignedURL2CdnURL(ctx, signURL, aliDomain)
	videoInternalDomain := getInternalDomain(ctx, TypeInternalGuideAli)
	liveInternalURL = alioss.SignedURL2CdnURL(ctx, signURL, videoInternalDomain)
	return
}
