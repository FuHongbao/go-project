/*
Package conf 用于项目基础配置。
*/
package conf

import (
	"flag"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/garyburd/redigo/redis"
	"log"
	"os"
	"path"
	"path/filepath"
	"xgit.xiaoniangao.cn/xngo/lib/xconf"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils/alists"
)

type Conf struct {
	Redis map[string]RedisConf `mapstructure:"redis"`

	//RedisCluster map[string]RedisClusterConf `mapstructure:"redisCluster"`

	Mongo map[string]MongoConf `mapstructure:"mongo"`

	// 基本配置
	App *struct {
		Name string `mapstructure:"name"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"app"`

	//日志输出
	Xlog *xlog.Config `mapstructure:"xlog"`

	LogConfig *xlog.Config `mapstructure:"logger"`

	TraceLogConfig *xlog.Config `mapstructure:"traceLog"`
	Speech         *struct {
		AccessKeyId     string `mapstructure:"access_key_id"`
		AccessKeySecret string `mapstructure:"access_key_secret"`
		AppKey          string `mapstructure:"app_key"`
		Domain          string `mapstructure:"domain"`
		Region          string `mapstructure:"region"`
		Url             string `mapstructure:"url"`
		Host            string `mapstructure:"host"`
	} `mapstructure:"speech"`
	// 各种名字或addr配置
	Addrs *struct {
		Ids string `mapstructure:"ids"`
	} `mapstructure:"addrs"`

	Bucket *struct {
		Album    string `mapstructure:"album"`
		Resource string `mapstructure:"resource"`
	} `mapstructure:"bucket"`

	Mq       *MqConf                      `mapstructure:"mq"`
	Sts      map[string]*alists.StsConfig `mapstructure:"sts"`
	Cdn      map[string]string            `mapstructure:"cdn"`
	Internal map[string]string            `mapstructure:"internal"`

	AliOSS map[string]struct {
		EndPoint        string `mapstructure:"endpoint"`
		Bucket          string `mapstructure:"bucket"`
		AccessKeyID     string `mapstructure:"access_key_id"`
		AccessKeySecret string `mapstructure:"access_key_secret"`
	} `mapstructure:"ali_oss"`
	CallBackUrl string `mapstructure:"callback_url"`
}

var (
	//全局配置
	C Conf

	// RDS 表示redis连接，key是业务名，value是redis连接
	RDS map[string]*RedisPool

	// RDSCluster 表示redis连接，key是业务名，value是redis cluster连接
	// RDSCluster XRedisClusterMap

	// DBS 表示mongo连接，key是db名，value是db连接
	DBS XMongoMap

	// 为true时只检查配置不启动服务
	CheckConfig = false

	// AliOSSClient 阿里云oss client
	AliOSSClient map[string]*oss.Client
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

// initAliOSSClient 初始化oss client，每次获取花费时间过长
func initAliOSSClient(c Conf) {
	AliOSSClient = map[string]*oss.Client{}
	for name, v := range c.AliOSS {
		if client, err := oss.New(v.EndPoint, v.AccessKeyID, v.AccessKeySecret); err != nil {
			log.Fatal(err)
		} else {
			AliOSSClient[name] = client
		}
	}
}

func init() {
	var env string
	flag.StringVar(&env, "env", "dev", "set env")
	var test bool
	flag.BoolVar(&test, "test", false, "set test case flag")
	flag.BoolVar(&CheckConfig, "checkconfig", false, "check config file")
	flag.Parse()
	Env = env

	initConfig(env)
	//lc.Init(1e5) // lc is not concurrent safe, user github.com/hashicorp/golang-lru instead

	initAliOSSClient(C)
	//初始化log
	xlog.Init(C.Xlog, C.App.Name)
	//InitLogger()
	//初始化mq
	InitMq(C.Mq)
	//初始化mongo
	DBS = GetMongoMap(C.Mongo)
	RDS = GetRedisMap(C.Redis)
}
