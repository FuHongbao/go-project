package alioss

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
)

// GetAlbumSignURL 获取影集sign url 域名不是cdn域名
func GetAlbumSignURL(ctx context.Context, key string, process int) (videoSignURL string) {
	videoSignURL, err := OssSignURLForAlbum(ctx, "album", key, getExpiredInSec(), process)
	if err != nil {
		xlog.ErrorC(ctx, "fail to get video sign url, key:%s, err:%v", key, err)
		return
	}
	videoSignURL = strings.ReplaceAll(videoSignURL, "%2F", "/")
	return
}
func getVideoSnapProcess(param string) string {
	return "video/snapshot," + param
}
func GetVideoSnapSignURL(ctx context.Context, key string, param string) (videoSnapSignURL string) {
	process := getVideoSnapProcess(param)
	videoSnapSignURL, err := OssSignURLForVideoSnap(ctx, "img", key, getExpiredInSec(), process)
	if err != nil {
		xlog.ErrorC(ctx, "fail to get video snap sign url, key:%s, err:%v", key, err)
		return
	}
	videoSnapSignURL = strings.ReplaceAll(videoSnapSignURL, "%2F", "/")
	return
}
func GetAlbumSnapSignURL(ctx context.Context, key string, param string) (albumSnapSignURL string) {
	process := getVideoSnapProcess(param)
	albumSnapSignURLBase, err := OssSignURLForVideoSnap(ctx, "album", key, getExpiredInSec(), process)
	if err != nil {
		xlog.ErrorC(ctx, "fail to get album snap sign url, key:%s, err:%v", key, err)
		return
	}
	albumSnapSignURL = strings.ReplaceAll(albumSnapSignURLBase, "%2F", "/")
	return
}
func GetLiveVideoSignURL(ctx context.Context, key string) (videoSignURL string) {
	videoSignURL, err := OssSignURL(ctx, "live", key, getExpiredInSec())
	if err != nil {
		xlog.ErrorC(ctx, "fail to get live video sign url, key:%s, err:%v", key, err)
		return
	}
	return
}

// 获取直播导播url
func GetLiveGuideSignURL(ctx context.Context, key string) (signURL string) {
	signURL, err := OssSignURL(ctx, "live_guide", key, getExpiredInSec())
	if err != nil {
		xlog.ErrorC(ctx, "fail to get sign url, key:%s, err:%v", key, err)
		return
	}
	return
}

//获取app日志url
func GetAppLogSignURL(ctx context.Context, key string) (videoSignURL string) {
	videoSignURL, err := OssSignURL(ctx, "app_log", key, getExpiredInSec())
	if err != nil {
		xlog.ErrorC(ctx, "fail to get live video sign url, key:%s, err:%v", key, err)
		return
	}
	return
}

// GetImageSignURL 获取图片地址
func GetImageSignURL(ctx context.Context, key string, actions []ImageAction) (signURL string) {
	signURL, err := OssSignURL(ctx, "img", key, getExpiredInSec(), actions...)
	if err != nil {
		xlog.ErrorC(ctx, "fail to get sign url, key:%s, err:%v", key, err)
		return
	}
	return
}

// GetImageSignURL 获取直播审核音乐地址
func GetAuditMusicSignURL(ctx context.Context, key string) (signURL string) {
	signURL, err := OssSignURL(ctx, "audit_music", key, getExpiredInSec())
	if err != nil {
		xlog.ErrorC(ctx, "fail to get sign url, key:%s, err:%v", key, err)
		return
	}
	return
}
func OssSignURLForAlbum(ctx context.Context, bucketName string, objName string, expiredInSec int64, process int) (signedURL string, err error) {
	c, ok := conf.C.AliOSS[bucketName]
	if !ok {
		return "", fmt.Errorf("fail get bucket conf, name:%s", bucketName)
	}
	var client *oss.Client
	if client, ok = conf.AliOSSClient[bucketName]; !ok {
		return "", fmt.Errorf("fail get bucket client, name:%s", bucketName)
	}
	var bucket *oss.Bucket
	if bucket, err = client.Bucket(c.Bucket); err != nil {
		return
	}
	if process > 0 {
		signedURL, err = bucket.SignURL(objName, oss.HTTPGet, expiredInSec, oss.Process("hls/sign"))
	} else {
		signedURL, err = bucket.SignURL(objName, oss.HTTPGet, expiredInSec)
	}
	xlog.DebugC(ctx, "bucketName:%s, objName:%s, expiredInSec:%s, url:%s", bucketName, objName, expiredInSec, signedURL)

	return
}
func OssSignURLForVideoSnap(ctx context.Context, bucketName string, objName string, expiredInSec int64, process string) (signedURL string, err error) {
	c, ok := conf.C.AliOSS[bucketName]
	if !ok {
		return "", fmt.Errorf("fail get bucket conf, name:%s", bucketName)
	}
	var client *oss.Client
	if client, ok = conf.AliOSSClient[bucketName]; !ok {
		return "", fmt.Errorf("fail get bucket client, name:%s", bucketName)
	}
	var bucket *oss.Bucket
	if bucket, err = client.Bucket(c.Bucket); err != nil {
		return
	}
	signedURL, err = bucket.SignURL(objName, oss.HTTPGet, expiredInSec, oss.Process(process))
	xlog.DebugC(ctx, "bucketName:%s, objName:%s, expiredInSec:%s, url:%s", bucketName, objName, expiredInSec, signedURL)

	return
}

// OssSignURL 生成oss的地址
func OssSignURL(ctx context.Context, bucketName string, objName string, expiredInSec int64, actions ...ImageAction) (signedURL string, err error) {
	//start := time.Now()
	c, ok := conf.C.AliOSS[bucketName]
	if !ok {
		return "", fmt.Errorf("fail get bucket conf, name:%s", bucketName)
	}
	//xlog.DebugC(ctx, "OssSignURL.get bucket use time:[%d]", time.Since(start))
	//start = time.Now()
	var client *oss.Client
	if client, ok = conf.AliOSSClient[bucketName]; !ok {
		return "", fmt.Errorf("fail get bucket client, name:%s", bucketName)
	}
	//xlog.DebugC(ctx, "OssSignURL.get client use time:[%d]", time.Since(start))
	//start = time.Now()
	var bucket *oss.Bucket
	if bucket, err = client.Bucket(c.Bucket); err != nil {
		return
	}

	options := ""
	for _, v := range actions {
		options += v.ToString()
	}
	//xlog.DebugC(ctx, "OssSignURL.get bucket use time:[%d]", time.Since(start))
	//start = time.Now()
	if options == "" {
		signedURL, err = bucket.SignURL(objName, oss.HTTPGet, expiredInSec)
	} else {
		signedURL, err = bucket.SignURL(objName, oss.HTTPGet, expiredInSec, oss.Process("image"+options))
	}
	//xlog.DebugC(ctx, "OssSignURL.SignURL use time:[%d]", time.Since(start))
	xlog.DebugC(ctx, "bucketName:%s, objName:%s, expiredInSec:%d, options:%s, url:%s", bucketName, objName, expiredInSec, options, signedURL)

	return
}

// SignedURL2CdnURL 替换网址
func SignedURL2CdnURL(ctx context.Context, signedURL, cdnDomain string) (cdnURL string) {
	u, err := url.Parse(signedURL)
	if err != nil {
		xlog.ErrorC(ctx, "fail to parse signedURL: %s, err: %v.", signedURL, err)
		return ""
	}
	u.Scheme = "https"
	u.Host = cdnDomain
	xlog.DebugC(ctx, "SignedURL2CdnURL.u :[%v]", u)
	return u.String()
}

// 获取过期时间
func getExpiredInSec() int64 {
	now := time.Now()
	nextMonthStart := getNextMonthStartSecondTime(now)
	if now.Unix()+86400 > nextMonthStart.Unix() {
		nextMonthStart = getNextMonthStartSecondTime(now.Add(time.Hour * 24 * 5))
	}
	return nextMonthStart.Unix() - now.Unix()
}

func getNextMonthStartSecondTime(d time.Time) time.Time {
	d = d.AddDate(0, 0, -d.Day()+1)
	thisMonthStartTime := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
	return thisMonthStartTime.Add(time.Hour*24*30*2 + 5*time.Second)
}
