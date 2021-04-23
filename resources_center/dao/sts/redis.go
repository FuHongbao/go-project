package sts

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils/alists"
)

const RedisName = "sts"

func getKey(name string) string {
	return fmt.Sprintf("upt:%s", name)
}

func GetUploadToken(ctx context.Context, name string) (resp *alists.StsResponse, ext int, err error) {
	key := getKey(name)
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}

	conn := pool.Get()
	ret, err := redis.Bytes(conn.Do("GET", key))
	if err != nil {
		if err == redis.ErrNil {
			err = nil
		}
		return
	}

	ext, _ = redis.Int(conn.Do("TTL", key))

	resp = &alists.StsResponse{}
	err = json.Unmarshal(ret, &resp)
	return
}

func SetUploadToken(ctx context.Context, name string, resp *alists.StsResponse, ext int) (err error) {
	key := getKey(name)
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}

	conn := pool.Get()

	value, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, err = redis.Bytes(conn.Do("SETEX", key, ext, value))
	if err != nil {
		return
	}

	return
}
