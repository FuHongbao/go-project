package conf

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/vmihailenco/msgpack"
	"time"
)

type RedisConf struct {
	Addr         string `mapstructure:"addr"`
	Auth         string `mapstructure:"auth"`
	Db           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"poolSize"`
	DialTimeout  int    `mapstructure:"dialTimeout"`
	ReadTimeout  int    `mapstructure:"readTimeout"`
	WriteTimeout int    `mapstructure:"writeTimeout"`
}

type RedisClusterConf struct {
	Addrs        []string `mapstructure:"addrs"`
	Auth         string   `mapstructure:"auth"`
	PoolSize     int      `mapstructure:"poolSize"`
	DialTimeout  int      `mapstructure:"dialTimeout"`
	ReadTimeout  int      `mapstructure:"readTimeout"`
	WriteTimeout int      `mapstructure:"writeTimeout"`
}

func GetRedisMap(confMap map[string]RedisConf) (RDS map[string]*RedisPool) {

	RDS = make(map[string]*RedisPool)
	for dbName, r := range confMap {
		dbName := dbName
		r := r
		RDS[dbName] = &RedisPool{redis.Pool{
			MaxIdle:     r.PoolSize, //这个配置项的poolSize是否对应maxidle？
			IdleTimeout: time.Minute,
			Dial: func() (conn redis.Conn, err error) {
				conn, err = redis.Dial("tcp", r.Addr,
					redis.DialReadTimeout(time.Duration(int64(r.ReadTimeout)*int64(time.Millisecond))),
					redis.DialConnectTimeout(time.Duration(int64(r.DialTimeout)*int64(time.Millisecond))),
					redis.DialWriteTimeout(time.Duration(int64(r.WriteTimeout)*int64(time.Millisecond))),
				)
				if err != nil {
					panic(fmt.Sprintf("redis.Dial err: %v, req: %v", err, r.Addr))
					return
				}

				if auth := r.Auth; auth != "" {
					if _, err = conn.Do("AUTH", auth); err != nil {
						panic(fmt.Sprintf("redis AUTH err: %v, req: %v,%v", err, r.Addr, auth))
						conn.Close()
						return
					}
				}
				if db := r.Db; db > 0 {
					if _, err = conn.Do("SELECT", db); err != nil {
						panic(fmt.Sprintf("redis SELECT err: %v, req: %v,%v", err, r.Addr, db))
						conn.Close()
						return
					}
				}
				return
			},
		},
		}
	}
	return
}

func GetRedisPool(redisName string) *RedisPool {
	return RDS[redisName]
}

// GetPoolRedis 获取pool
func GetPoolRedis(name string) (pool *RedisPool, err error) {
	if p, ok := RDS[name]; ok {
		pool = p
		return
	}
	return nil, fmt.Errorf("fail to get redis[%s] config", name)
}

// Get cache from redis.
func (rp *RedisPool) GetString(key string) (string, error) {
	conn := rp.Get()
	defer conn.Close()
	v, err := conn.Do("GET", key)
	if err != nil {
		return "", err
	}
	if v == nil {
		return "", nil
	}
	return redis.String(v, err)
}

// Get cache from redis.
func (rp *RedisPool) GetPrimitiveInterface(key string) (interface{}, error) {
	conn := rp.Get()
	defer conn.Close()
	v, err := conn.Do("GET", key)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// GetMulti get cache from redis.
func (rp *RedisPool) GetMulti(keys []string) ([]interface{}, error) {
	conn := rp.Get()
	defer conn.Close()
	values, err := redis.Values(conn.Do("MGET", keys))
	if err != nil {
		return nil, err
	}
	return values, nil
}

// Put put cache to redis.
func (rp *RedisPool) PutPrimitive(key string, val interface{}) error {
	conn := rp.Get()
	defer conn.Close()
	_, err := conn.Do("SET", key, val)
	return err
}

// Put put cache to redis.
func (rp *RedisPool) PutPrimitiveExpire(key string, val interface{}, timeout time.Duration) error {
	conn := rp.Get()
	defer conn.Close()
	_, err := conn.Do("SETEX", key, int64(timeout/time.Second), val)
	return err
}

// Get cache from redis.
func (rp *RedisPool) GetStruct(key string, result interface{}) (err error, exists bool) {
	reply, err := rp.GetString(key)
	if err != nil || reply == "" {
		return err, false
	}
	err = msgpack.Unmarshal([]byte(reply), result)
	return err, true
}

// Put put cache to redis.
func (rp *RedisPool) PutStruct(key string, val interface{}) error {
	data, err := msgpack.Marshal(val)
	if err != nil {
		return err
	}
	err = rp.PutPrimitive(key, string(data))
	return err
}

// Put put cache to redis.
func (rp *RedisPool) PutStructExpire(key string, val interface{}, timeout time.Duration) error {
	data, err := msgpack.Marshal(val)
	if err != nil {
		return err
	}
	err = rp.PutPrimitiveExpire(key, string(data), timeout)
	return err
}

// Delete delete cache in redis.
func (rp *RedisPool) Delete(key string) error {
	conn := rp.Get()
	defer conn.Close()
	_, err := conn.Do("DEL", key)
	return err
}

// IsExist check cache's existence in redis.
func (rp *RedisPool) IsExist(key string) (bool, error) {
	conn := rp.Get()
	defer conn.Close()
	return redis.Bool(conn.Do("EXISTS", key))
}

// Incr increase counter in redis.
func (rp *RedisPool) Incr(key string, incrBy int) (int64, error) {
	conn := rp.Get()
	defer conn.Close()
	return redis.Int64(conn.Do("INCRBY", key, incrBy))
}

// Decr decrease counter in redis.
func (rp *RedisPool) Decr(key string) (int64, error) {
	conn := rp.Get()
	defer conn.Close()
	return redis.Int64(conn.Do("DECR", key))
}

// expire key
func (rp *RedisPool) Expire(key string, timeout time.Duration) error {
	conn := rp.Get()
	defer conn.Close()
	_, err := conn.Do("expire", key, int64(timeout/time.Second))
	return err
}

//cluster的需要另加
