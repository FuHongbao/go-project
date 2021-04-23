package snap

import (
	"context"
	"fmt"
	"strconv"
	"xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo/bson"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByQid"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/resource"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils/alioss"
)

func UpdateResCover(ctx context.Context, resId, snapId int64) (ret bool, err error) {
	doc, err := DaoByQid.GetDocByQid(resId)
	if err != nil {
		xlog.ErrorC(ctx, "UpdateResCover.GetDocByQid failed, err:[%v]", err)
		return
	}
	resKey := strconv.FormatInt(resId, 10)
	if doc == nil {
		//xlog.DebugC(ctx, "UpdateResCover.GetDocByQid not found, key:[%v]", resId)
		doc, err = resource.GetCache(ctx, resKey)
		if err != nil {
			return
		}
		if doc == nil {
			//xlog.DebugC(ctx, "UpdateResCover.GetCache not found, key:[%v]", resKey)
			return
		}
	}
	doc.Cover = snapId
	err = resource.SetCache(ctx, resKey, doc, api.CacheShortTime)
	if err != nil {
		xlog.ErrorC(ctx, "UpdateResCover.SetCache failed, err:[%v]", err)
		return
	}
	qry := bson.M{"_id": resId}
	updata := bson.M{"$set": bson.M{"cover": snapId}}
	err = DaoByQid.UpdateResourceDoc(resId, qry, updata)
	if err != nil {
		xlog.ErrorC(ctx, "UpdateResCover.UpdateResourceDoc failed, err:[%v], qry:[%v], update:[%v]", err, qry, updata)
		return
	}
	xlog.DebugC(ctx, "UpdateResCover resID:[%d], snapID:[%d]", resId, snapId)
	ret = true
	return
}

func GetVideoFrameUrls(ctx context.Context, key string, cnt int, startTime, spaceTime int64, w, h int) (urls []string) {
	for i := 0; i < cnt; i++ {
		t := int64(i)*spaceTime + startTime
		process := fmt.Sprintf("t_%d,f_jpg,w_%d,h_%d", t, w, h)
		url := alioss.GetVideoSnapSignURL(ctx, key, process)
		urls = append(urls, url)
	}
	return
}
func GetAlbumFrameUrls(ctx context.Context, key string, cnt int, startTime, spaceTime int64, w, h int) (urls []string) {
	for i := 0; i < cnt; i++ {
		t := int64(i)*spaceTime + startTime
		process := fmt.Sprintf("t_%d,f_jpg,w_%d,h_%d,ar_auto", t, w, h)
		url := alioss.GetAlbumSnapSignURL(ctx, key, process)
		urls = append(urls, url)
	}
	return
}
