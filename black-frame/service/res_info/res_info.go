package res_info

import (
	"context"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/api"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/conf"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/util/net"
)

const (
	//RedisName      = "resources"
	//CacheShortTime = 5 * 60
	ServiceName             = "resources-center"
	ServiceReplaceCoverPath = "snap/replace_snap"
)

/*
func getKey(id string) string {
	return fmt.Sprintf("res:%s", id)
}
func SetCache(ctx context.Context, id string, doc *api.XngResourceInfoDoc, ext int) (err error) {
	key := getKey(id)
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}

	conn := pool.Get()

	value, err := json.Marshal(doc)
	if err != nil {
		return
	}
	_, err = redis.Bytes(conn.Do("SETEX", key, ext, value))
	if err != nil {
		return
	}
	return
}

func UpdateResCover(ctx context.Context, resId, snapId int64) (err error) {
	doc, err := dao_by_qid.GetDocByQid(resId)
	if err != nil {
		xlog.ErrorC(ctx, "DelBlackFrame.GetDocByQid failed, err:[%v]", err)
		return
	}
	doc.Cover = snapId
	xlog.DebugC(ctx, "UpdateResCover resID:[%d], snapID:[%d]", resId, snapId)
	resKey := strconv.FormatInt(resId, 10)
	err = SetCache(ctx, resKey, doc, CacheShortTime)
	if err != nil {
		xlog.ErrorC(ctx, "DelBlackFrame.SetCache failed, err:[%v]", err)
		return
	}
	qry := bson.M{"_id": resId}
	updata := bson.M{"$set": bson.M{"cover": snapId}}
	err = dao_by_qid.UpdateResourceDoc(resId, qry, updata)
	if err != nil {
		xlog.ErrorC(ctx, "DelBlackFrame.UpdateResourceDoc failed, err:[%v], qry:[%v], update:[%v]", err, qry, updata)
		return
	}
	return
}
*/

type ReqReplaceSnap struct {
	Key   string `json:"key"`
	Cover string `json:"cover"`
}
type RespReplaceSnap struct {
	Status int `json:"status"`
}

type RespReplaceSnapData struct {
	Ret  int             `json:"ret"`
	Data RespReplaceSnap `json:"data"`
}

func UpdateResCoverByResCenter(ctx context.Context, resKey, snapKey string) (ret int, err error) {
	req := ReqReplaceSnap{
		Key:   resKey,
		Cover: snapKey,
	}
	resp := RespReplaceSnapData{}
	if conf.Env == lib.PROD {
		err = net.XngServiceCallPostWithRetry(ctx, &net.Consul{}, ServiceName, ServiceReplaceCoverPath, req, &resp, time.Second*2, 1)
		if err != nil {
			return
		}
	} else {
		err = net.Post(ctx, api.ReplaceCoverURL, time.Second*2, req, &resp)
		if err != nil {
			return
		}
	}
	ret = resp.Data.Status
	return
}
