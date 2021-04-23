package url

import (
	"context"
	"github.com/gin-gonic/gin"
	url2 "net/url"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	resourceDao "xgit.xiaoniangao.cn/xngo/service/resources_center/dao/resource"
	urlService "xgit.xiaoniangao.cn/xngo/service/resources_center/service/url"
)

type ReqAlbumM3u8ExpUrl []struct {
	ID       string `json:"id" binding:"required"`
	IsStream int    `json:"is_stream"`
}

type RespAlbumM3u8ExpUrl struct {
	URLs map[string]RespURLWithM3u8HitExp `json:"urls"`
}

// AlbumM3u8ExpURL 小程序m3u8链接播放实验，使用新cdn域名
func AlbumM3u8ExpURL(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req ReqAlbumM3u8ExpUrl
	if !xc.GetReqObject(&req) {
		return
	}
	resp := &RespAlbumM3u8ExpUrl{URLs: make(map[string]RespURLWithM3u8HitExp)}

	for _, v := range req {
		if v.ID == "" {
			continue
		}
		key := v.ID
		url, urlInternal := urlService.GetAlbumURL(xc, key, 0)
		var urlM3u8, urlM3u8Internal string
		switch v.IsStream {
		case 1:
			key += "/index_0.m3u8"
		case 2:
			key += "/index.m3u8"
		}
		if v.IsStream != 0 {
			urlM3u8, urlM3u8Internal = urlService.GetAlbumURL(xc, key, v.IsStream)
		}
		hit, url, urlM3u8 := replaceExpHitHost(xc, v.ID, url, urlM3u8)
		resp.URLs[v.ID] = RespURLWithM3u8HitExp{URL: url, URLInternal: urlInternal, URLM3u8: urlM3u8, URLM3u8Internal: urlM3u8Internal, IsHit: hit}
	}
	xc.ReplyOK(resp)
}

func replaceExpHitHost(ctx context.Context, key, url, urlM3u8 string) (ret bool, retUrl string, retUrlM3u8 string) {
	retUrl = url
	retUrlM3u8 = urlM3u8
	hit, err := resourceDao.ExistsM3u8ExpID(ctx, key)
	if err != nil {
		return
	}
	if !hit {
		return
	}
	ret = hit
	u, err := url2.Parse(url)
	if err != nil {
		return
	}
	u.Scheme = "https"
	u.Host = "cdn-xalbum-mp4.xiaoniangao.cn"
	retUrl = u.String()
	if urlM3u8 != "" {
		u2, err := url2.Parse(urlM3u8)
		if err != nil {
			return
		}
		u2.Scheme = "https"
		u2.Host = "cdn-xalbum-m3u8.xiaoniangao.cn"
		retUrlM3u8 = u2.String()
	}
	return
}
