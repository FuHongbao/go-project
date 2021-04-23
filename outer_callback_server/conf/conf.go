/*
Package conf 用于项目基础配置。
*/
package conf

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"path"
	"path/filepath"
	"xgit.xiaoniangao.cn/xngo/lib/lc"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xconf"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
)

//MQ defined
type MQ struct {
	NameSrvAddr string `yaml:"nameSrvAddr"`
}

//Conf defined
type Conf struct {

	// 基本配置
	App *struct {
		Name string `yaml:"name"`
		Port int    `yaml:"port"`
	} `yaml:"app"`

	//日志输出
	Xlog *xlog.Config `yaml:"xlog"`
	//日志上报
	Bizlog *xlog.Config `yaml:"bizlog"`
	Mqlog  *xlog.Config `yaml:"mqlog"`
	// 各种名字或addr配置
	Addrs *struct {
		Ids        string `yaml:"ids"`
		UserCenter string `yaml:"usercenter"`
	} `yaml:"addrs"`

	//MQ 相关配置
	MQ    map[string]MQ        `yaml:"mq"`
	MqGo  MqConf               `mapstructure:"go_mq"`
	Mongo map[string]MongoConf `yaml:"mongo"`

	Redis map[string]RedisConf `yaml:"redis"`

	//微信账号
	Wxids map[string]string `yaml:"wxids"`

	//微信token
	WxToken map[string]string `yaml:"wxtoken"`

	PhpReceiveAddr map[string]string `yaml:"phpReceiveAddr"`

	MaGrayMsgTypes []string `yaml:"maGrayMsgTypes"`

	MaGrayOpenids []string `yaml:"maGrayOpenids"`

	MpGrayMsgTypes []string `yaml:"mpGrayMsgTypes"`

	MpGrayOpenids []string `yaml:"mpGrayOpenids"`

	SubGrayMsgTypes []string `yaml:"subGrayMsgTypes"`

	SubGrayOpenids []string `yaml:"subGrayOpenids"`
}

var (
	// Gin 全局的gin
	Gin *gin.Engine

	//C 全局配置
	C Conf

	// CheckConfig 为true时只检查配置不启动服务
	CheckConfig = false

	// DBS 表示mongo连接，key是db名，value是db连接
	DBS XMongoMap

	// RDS 表示redis连接，key是业务名，value是redis连接
	RDS XRedisMap
	// Env defined
	Env string
	//Bizlog defined
	Bizlog *xlog.XLogger
)

func initGin() {
	Gin = gin.New()
	Gin.Use(gin.Recovery())
	Gin.Use(xng.Boss())
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
	flag.StringVar(&env, "env", "dev", "set env")
	var test bool
	flag.BoolVar(&test, "test", false, "set test case flag")
	flag.BoolVar(&CheckConfig, "checkconfig", false, "check config file")
	flag.Parse()

	initConfig(env)
	initGin()
	lc.Init(1e5)

	gin.SetMode(gin.ReleaseMode)
	xlog.Init(C.Xlog, C.App.Name)
	Bizlog = xlog.NewLogger(C.Bizlog, C.App.Name)
	//初始化mongo
	DBS = GetMongoMap(C.Mongo)
	RDS = GetRedisMap(C.Redis)
	InitMq(C.MqGo, C.Mqlog)
	Env = env
}
