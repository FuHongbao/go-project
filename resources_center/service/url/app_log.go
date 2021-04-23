package url

import (
	"context"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils/alioss"
)

// GetAlbumURL 获取直播视频播放地址
func GetAppLogURL(ctx context.Context, key string) (url string, InternalURL string) {
	url = alioss.GetAppLogSignURL(ctx, key)
	InternalDomain := getInternalDomain(ctx, TypeInternalAppLog)
	InternalURL = alioss.SignedURL2CdnURL(ctx, url, InternalDomain)
	return
}
