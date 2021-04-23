package url

import (
	"context"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"time"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils/alioss"
)

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
	return thisMonthStartTime.Add(time.Hour*24*30 + 5*time.Second)
}

func GetStaticImageOssURL(ctx context.Context, filename, qs string, bucket *oss.Bucket) (imageURL string, err error) {
	actions := alioss.ParseQs(qs)
	options := ""
	for _, v := range actions {
		options += v.ToString()
	}

	if options == "" {
		imageURL, err = bucket.SignURL(filename, oss.HTTPGet, getExpiredInSec())
		if err != nil {
			return
		}
	} else {
		imageURL, err = bucket.SignURL(filename, oss.HTTPGet, getExpiredInSec(), oss.Process("image"+options))
		if err != nil {
			return
		}
	}
	return
}
func GetStaticImageURL(ctx context.Context, filename, qs string, bucket *oss.Bucket) (imageURL string, err error) {
	signUrl, err := GetStaticImageOssURL(ctx, filename, qs, bucket)
	if err != nil {
		return
	}
	domain := getCDNDomain(ctx, "static_ali")
	imageURL = alioss.SignedURL2CdnURL(ctx, signUrl, domain)
	return
}

func GetStaticURL(ctx context.Context, objectKey string) (url, urlInternal string) {
	domain := getCDNDomain(ctx, "static_ali")
	url = fmt.Sprintf("https://%s/%s", domain, objectKey)
	internalDomain := getInternalDomain(ctx, TypeInternalStaticAli)
	urlInternal = fmt.Sprintf("https://%s/%s", internalDomain, objectKey)
	return
}
