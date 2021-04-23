/*
Package conf 用于项目基础配置。
*/
package conf

import (
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"path"
	"path/filepath"
	"xgit.xiaoniangao.cn/xngo/lib/xconf"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
)

type Conf struct {
	// 基本配置
	App *struct {
		Name string `mapstructure:"name"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"app"`

	//日志输出
	Xlog *xlog.Config `mapstructure:"xlog"`
	Mq   *MqConf      `mapstructure:"mq"`
	// 各种名字或addr配置
	Addrs *struct {
		Clog string `mapstructure:"clog"`
		Ids  string `mapstructure:"ids"`
	} `mapstructure:"addrs"`
	Mongo map[string]MongoConf `mapstructure:"mongo"`
	Redis map[string]RedisConf `mapstructure:"redis"`
}

var (

	//全局的gin
	Gin *gin.Engine

	//全局配置
	C Conf

	// RDS 表示redis连接，key是业务名，value是redis连接
	RDS map[string]*RedisPool

	// DBS 表示mongo连接，key是db名，value是db连接
	DBS XMongoMap
	// Env 环境
	Env string
)

type RedisPool struct {
	redis.Pool
}

func initConfig(env string) {
	configDir := getConfDir()
	configName := fmt.Sprintf("%s.conf.yaml", env)
	configPath := path.Join(configDir, configName)

	C = Conf{}
	err := xconf.LoadConfig(configPath, &C)
	if err != nil {
		log.Fatal(err)
	}
}
func getConfDir() string {
	dir := "conf"
	for i := 0; i < 3; i++ {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			break
		}
		dir = filepath.Join("..", dir)
	}

	return dir
}

func init() {
	var env string
	flag.StringVar(&env, "env", "test", "set env")
	var test bool
	flag.BoolVar(&test, "test", false, "set test case flag")
	flag.Parse()
	Env = env
	initConfig(env)

	//initRedis(&C)

	//lc.Init(1e5)

	// xlog 初始化
	xlog.Init(C.Xlog, C.App.Name)
	InitMq(C.Mq)
	//初始化mongo
	//DBS = GetMongoMap(C.Mongo)
	//RDS = GetRedisMap(C.Redis)
	//初始化redis
	//xng.InitRedis(RDS)
}
