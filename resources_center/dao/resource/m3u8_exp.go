package resource

import (
	"context"
	"github.com/garyburd/redigo/redis"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
)

const (
	RedisM3u8Exp = "resources"
	SetM3u8Exp   = "m3u8_exp_set"
)

func getM3u8ExpKey(id string) string {
	return "m3u8:" + id
}
func SetM3u8ExpID(ctx context.Context, ids []string) (err error) {
	var keys []interface{}
	keys = append(keys, SetM3u8Exp)
	for _, id := range ids {
		key := getM3u8ExpKey(id)
		keys = append(keys, key)
	}
	pool, err := conf.GetPoolRedis(RedisM3u8Exp)
	if err != nil {
		return
	}
	conn := pool.Get()

	_, err = conn.Do("SADD", keys...)
	if err != nil {
		return
	}
	return
}

func ExistsM3u8ExpID(ctx context.Context, id string) (exists bool, err error) {
	key := getM3u8ExpKey(id)
	pool, err := conf.GetPoolRedis(RedisM3u8Exp)
	if err != nil {
		return
	}
	conn := pool.Get()
	ret, err := redis.Int(conn.Do("SISMEMBER", SetM3u8Exp, key))
	if err != nil {
		return
	}
	if ret == 1 {
		exists = true
	}
	return
}
