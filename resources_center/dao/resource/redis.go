package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"strconv"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
)

const (
	RedisName         = "resources"
	MultiAbortRecord  = "multi_abort_record"
	RunningRoutineKey = "RunRoutine"
)

func getKey(id string) string {
	return fmt.Sprintf("res:%s", id)
}

func GetCache(ctx context.Context, id string) (resp *api.XngResourceInfoDoc, err error) {
	key := getKey(id)
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
	//xlog.DebugC(ctx, "success get doc by cache:[%s]", string(ret))
	resp = &api.XngResourceInfoDoc{}
	err = json.Unmarshal(ret, &resp)
	return
}
func GetCacheBatch(ctx context.Context, ids []string) (qDocs map[string]*api.XngResourceInfoDoc, idsNotFound []string, err error) {
	var keys []interface{}
	for _, id := range ids {
		keys = append(keys, getKey(id))
	}
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}
	conn := pool.Get()
	reply, err := conn.Do("MGET", keys...)
	if err != nil {
		return
	}
	rets, err := redis.ByteSlices(reply, err)
	if err != nil && err != redis.ErrNil {
		xlog.ErrorC(ctx, "GetCacheBatch.get cache failed", map[string]interface{}{"error": err, "args": rets})
		return
	}
	qDocs = map[string]*api.XngResourceInfoDoc{}
	for i, ret := range rets {
		if ret == nil { //资源不存在
			idsNotFound = append(idsNotFound, ids[i])
			//qDocs[ids[i]] = nil
			continue
		}
		doc := &api.XngResourceInfoDoc{}
		errIgnore := json.Unmarshal(ret, &doc)
		if errIgnore != nil {
			err = errIgnore
			return
		}
		qDocs[ids[i]] = doc
	}
	return
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

func GetQeTagKeyName(key string) string {
	return fmt.Sprintf("QeTagCache:%s", key)
}
func SetQeTagCache(key string, value string, ext int) (err error) {
	keyName := GetQeTagKeyName(key)
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		_, err = conn.Do("SETEX", keyName, ext, value)
		if err != nil {
			continue
		}
		break
	}
	return
}

func GetQeTagCache(key string) (value string, err error) {
	keyName := GetQeTagKeyName(key)
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}

	conn := pool.Get()
	value, err = redis.String(conn.Do("GET", keyName))
	if err != nil {
		if err == redis.ErrNil {
			err = nil
			return
		}
	}
	return
}

func GetKeyName(key string) string {
	return fmt.Sprintf("MultiUpload:%s", key)
}

func AddMultiUploadRecord(key string, value *api.MultiUploadRecord) (err error) {
	keyName := GetKeyName(key)
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		bts, err := json.Marshal(value)
		if err != nil {
			continue
		}
		_, err = conn.Do("SET", keyName, bts)
		if err != nil {
			continue
		}
		break
	}
	return
}

func ExitMultiUploadRecord(key string) (exists bool, err error) {
	keyName := GetKeyName(key)
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		exists, err = redis.Bool(conn.Do("EXISTS", keyName))
		if err != nil {
			continue
		}
		break
	}
	return
}

func GetMultiUploadRecord(key string) (value *api.MultiUploadRecord, err error) {
	keyName := GetKeyName(key)
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}
	conn := pool.Get()
	bts, err := redis.Bytes(conn.Do("GET", keyName))
	if err != nil {
		if err == redis.ErrNil {
			err = nil
			return
		}
		return
	}
	err = json.Unmarshal(bts, &value)
	return
}

func DelMultiUploadRecord(key string) (err error) {
	keyName := GetKeyName(key)
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		_, err = conn.Do("DEL", keyName)
		if err != nil {
			continue
		}
		break
	}
	return
}

func AddMultiAbortRecord(value *api.MultiAbortRecord) (err error) {
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}

	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		score := time.Now().Unix()
		bts, err := json.Marshal(value)
		if err != nil {
			continue
		}
		_, err = conn.Do("ZADD", MultiAbortRecord, score, bts)
		if err != nil {
			continue
		}
		break
	}
	return
}

func DelMultiAbortRecord(value *api.MultiAbortRecord) (err error) {
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		bts, err := json.Marshal(value)
		if err != nil {
			continue
		}
		_, err = conn.Do("ZREM", MultiAbortRecord, bts)
		if err != nil {
			continue
		}
		break
	}
	return
}

func DelRangeMultiAbortRecord(score int64) (err error) {
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		_, err = conn.Do("ZREMRANGEBYSCORE", MultiAbortRecord, 1, score)
		if err != nil {
			continue
		}
		break
	}
	return
}

func GetRangMultiAbortRecord(score int64) (values []*api.MultiAbortRecord, err error) {
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		bts, err := redis.ByteSlices(conn.Do("ZRANGEBYSCORE", MultiAbortRecord, 1, score))
		if err != nil {
			continue
		}
		for _, bt := range bts {
			var value api.MultiAbortRecord
			err = json.Unmarshal(bt, &value)
			if err != nil {
				break
			}
			values = append(values, &value)
		}
		break
	}
	return
}

func SetRunningRoutine() (ok bool, err error) {
	pool, err := conf.GetPoolRedis(RedisName)
	if err != nil {
		return
	}
	conn := pool.Get()
	now := time.Now().Unix()
	value := strconv.FormatInt(now, 10)
	ret, err := redis.String(conn.Do("SET", RunningRoutineKey, value, "EX", 300, "NX"))
	if err != nil {
		if err == redis.ErrNil {
			ok = false
			err = nil
		}
		return
	}
	if ret == "OK" {
		ok = true
	} else {
		ok = false
	}
	return
}
