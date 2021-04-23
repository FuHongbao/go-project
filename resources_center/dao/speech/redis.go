package speech

import (
	"context"
	"github.com/garyburd/redigo/redis"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
)

const (
	RedisName     = "sph_token"
	RedisTokenKey = "speech_token"
)

func getTokenKey() string {
	return RedisTokenKey
}
func GetSpeechToken(ctx context.Context) (token string, ext int, err error) {
	key := getTokenKey()
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}

	conn := pool.Get()
	token, err = redis.String(conn.Do("GET", key))
	if err != nil {
		if err == redis.ErrNil {
			err = nil
		}
		return
	}
	ext, _ = redis.Int(conn.Do("TTL", key))
	return
}

func SetSpeechToken(ctx context.Context, token string, ext int) (err error) {
	key := getTokenKey()
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}
	conn := pool.Get()
	_, err = redis.Bytes(conn.Do("SETEX", key, ext, token))
	if err != nil {
		return
	}
	xlog.DebugC(ctx, "SetSpeechToken set key:[%s], value:[%s], ext:[%d]", key, token, ext)
	return
}
