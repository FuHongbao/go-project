package conf

import (
	"log"
	"xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo"
)

//XMongoMap defined
type XMongoMap map[string]*mgo.Session

//MongoConf defined
type MongoConf struct {
	Dsn string `yaml:"dsn"`
}

//GetMongoMap defined
func GetMongoMap(confMap map[string]MongoConf) XMongoMap {
	xMongoMap := XMongoMap{}
	for db, m := range confMap {
		session, err := mgo.Dial(m.Dsn)
		if err != nil {

			log.Fatalf("mongo [dsn: %s]  load fail: %v", m.Dsn, err)
		}
		session.SetMode(mgo.SecondaryPreferred, true)
		xMongoMap[db] = session
	}
	return xMongoMap
}
