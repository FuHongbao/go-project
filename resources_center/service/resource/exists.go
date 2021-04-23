package resource

import (
	"context"
)

func CheckResExists(ctx context.Context, key string, kind int) (isExist bool, err error) {
	stsName := GetMtsStsName(kind)
	ossInfo, err := GetAliOssClient(ctx, stsName)
	if err != nil {
		return
	}
	bucket, err := ossInfo.Client.Bucket(ossInfo.Sts.Bucket)
	if err != nil {
		return
	}
	isExist, err = bucket.IsObjectExist(key)
	if err != nil {
		return
	}
	return
}
