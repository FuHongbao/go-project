package conf

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"os"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
)

//RedisConf defined
type RedisConf struct {
	Addr string `yaml:"addr"`
	Auth string `yaml:"auth"`
	Db   int    `yaml:"db"`
}

//XRedisMap defined
type XRedisMap map[string]*redis.Pool

//GetRedisMap defined
func GetRedisMap(confMap map[string]RedisConf) XRedisMap {
	var xRedisMap = XRedisMap{}
	for db, r := range confMap {
		xlog.Debug("GetRedisMap.db:[%v], r:[%v]", db, r)
		rd := &redis.Pool{
			MaxIdle:     30,
			IdleTimeout: time.Minute,
			Dial: func() (conn redis.Conn, err error) {
				conn, err1 := redis.Dial("tcp", r.Addr,
					redis.DialReadTimeout(time.Second),
					redis.DialConnectTimeout(time.Second),
				)
				if err1 != nil {
					err = fmt.Errorf("redis.Dial err: %v, req: %v", err1, r.Addr)
					log.Println(err)
					xlog.Error(err.Error())
					os.Exit(1)
					return
				}

				if auth := r.Auth; auth != "" {
					if _, authErr := conn.Do("AUTH", auth); err1 != nil {
						err = fmt.Errorf("redis AUTH err: %v, req: %v,%v", authErr, r.Addr, auth)
						log.Println(err)
						xlog.Error(err.Error())
						err = conn.Close()
						if err != nil {
							log.Println(err)
							xlog.Error(err.Error())
						}
						os.Exit(1)
						return
					}
				}
				if db := r.Db; db > 0 {
					if _, dbError := conn.Do("SELECT", db); dbError != nil {
						err = fmt.Errorf("redis SELECT err: %v, req: %v,%v", dbError, r.Addr, db)
						log.Println(err)
						xlog.Error(err.Error())
						err = conn.Close()
						if err != nil {
							log.Println(err)
							xlog.Error(err.Error())
						}
						os.Exit(1)
						return
					}
				}
				return
			},
		}
		xRedisMap[db] = rd
	}
	return xRedisMap
}
