package videoRedis

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
)

const (
	XngResAccountRedis = "account_redis" //用户的资源引用计数, 与本上传功能无关，按原逻辑执行的
	XngAlbumRedis      = "album"         //推送影集制作消息的redis
	//XngTransRecord     = "trans_pool"         //临时转码记录，将转码中的资源记录临时添加到此处，以qid为键，jobID为值，用于重复校验
	//XngTransCallBack   = "trans_callback"     //以jobID为键，qid为值又临时存储了一份，用于消息回调时使用
	//XngUploadRecord    = "uploading_pool"     //上传中资源的临时记录，用于防止重复上传
	//XngUserDocUpdate   = "update_user_resdoc" //用于阿里云回调时更新user资源记录
	ResourcesRedis = "resources"

	XngAlbumSuccessQueue = "tpl_album_over_notify_list" //key名，通知制作成功的list
	XngAlbumFailQueue    = "tpl_album_list_fail"        //key名，通知制作失败的list
	UserResourceZset     = "zset_u_imgs"                //key名，用户的资源引用计数, 与本上传功能无关，按原逻辑执行的

	HandleResInfoLifeTime = 300
	UploadRecordLifeTime  = 3600  //上传中状态的生命周期为3600秒
	TransRecordLifeTime   = 86400 //转码中状态的生命周期为86400秒
	LifeTimeNotSet        = -1    //代表key未设置生命周期
)

//增加引用计数
func IncreaseUserAccount(mid int64) (err error) {
	pool := conf.GetRedisPool(XngResAccountRedis)
	if pool == nil {
		err = errors.New("get redis pool error")
		return
	}
	conn := pool.Get()
	_, err = conn.Do("zIncrBy", UserResourceZset, 1, mid)
	if err != nil {
		return
	}
	return nil
}

//是否正在上传中
func IsUploading(key string) (exist bool, err error) {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		err = errors.New("func IsUploading get redis pool is nil")
		return
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		exist, err = redis.Bool(conn.Do("EXISTS", key))
		if err != nil {
			continue
		}
		break
	}
	return
}

//添加上传中的资源记录
func AddUploadingRecord(key string, value int64) (err error) {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		return errors.New("AddUploadingRecord get redis pool error")
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		_, err = conn.Do("SETEX", key, UploadRecordLifeTime, value)
		if err != nil {
			continue
		}
		break
	}
	return
}

//删除上传中的资源记录
func DelUploadingRecord(key string) (err error) {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		return errors.New("DelUploadingRecord get redis pool error")
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		_, err = conn.Do("DEL", key)
		if err != nil {
			continue
		}
		break
	}
	return
}

func getTransKey(qid int64) string {
	return fmt.Sprintf("transCode:%d", qid)
}

//是否正在转码中
func IsTransCoding(key int64) (exist bool, err error) {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		err = errors.New("IsTransCoding get redis pool error")
		return
	}
	keyName := getTransKey(key)
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		exist, err = redis.Bool(conn.Do("EXISTS", keyName))
		if err != nil {
			continue
		}
		break
	}
	return
}

//用于同一时间段重复提交转码的校验
func AddResourceTransRecord(key int64, value string) (err error) {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		return errors.New("AddResourceTransRecord get redis pool error")
	}
	keyName := getTransKey(key)
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		_, err = conn.Do("SETEX", keyName, TransRecordLifeTime, value)
		if err != nil {
			continue
		}
		break
	}
	return
}

func DelResourceTransRecord(key int64) error {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		return errors.New("get redis pool error")
	}
	keyName := getTransKey(key)
	conn := pool.Get()
	_, err := conn.Do("DEL", keyName)
	if err != nil {
		return err
	}
	return nil
}

//用于结果回调时更新信息
func AddTransJobRecord(key string, value int64) (err error) {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		return errors.New("AddTransJobRecord get redis pool error")
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		_, err := conn.Do("SETEX", key, TransRecordLifeTime, value)
		if err != nil {
			continue
		}
		break
	}
	return nil
}

func GetTransJobRecord(key string) (qid int64, err error) {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		return 0, errors.New("GetTransJobRecord get redis pool error")
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		qid, err = redis.Int64(conn.Do("GET", key))
		if err != nil {
			continue
		}
		break
	}
	return
}

func DelTransJobRecord(key string) error {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		return errors.New("get redis pool error")
	}
	conn := pool.Get()
	_, err := conn.Do("DEL", key)
	if err != nil {
		return err
	}
	return nil
}

func AddAlbumToTransList(key string, value int64) error {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		return errors.New("get redis pool error")
	}
	conn := pool.Get()
	_, err := conn.Do("LPUSH", key, value)
	if err != nil {
		return err
	}
	exist, err := redis.Int(conn.Do("TTL", key))
	if err != nil {
		xlog.Error("ttl life time error, qid:%s, err:%v", key, err)
		return err
	}
	if exist == LifeTimeNotSet {
		_, err = conn.Do("EXPIRE", key, TransRecordLifeTime) //时间放长，一天
		if err != nil {
			xlog.Error("set life time for key error, qid:%s, error:%v", key, err)
			return err
		}
	}

	return nil
}

func GetTransListCnt(key string) (cnt int, err error) {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		return 0, errors.New("GetTransListCnt get redis pool error")
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		cnt, err = redis.Int(conn.Do("LLEN", key))
		if err != nil {
			continue
		}
		break
	}
	return
}

func GetAidsFromTransList(key string) (aids []int64, err error) {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		return aids, errors.New("get redis pool error")
	}
	conn := pool.Get()
	aids, err = redis.Int64s(conn.Do("LRANGE", key, 0, -1))
	if err != nil {
		return
	}
	return
}

func DelAidsFromTransList(key string) error {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		return errors.New("get redis pool error")
	}
	conn := pool.Get()
	_, err := conn.Do("DEL", key)
	if err != nil {
		return err
	}
	return nil
}

func GetResIdsFromList(key string) (resIdMidStrings []string, err error) {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		err = errors.New("GetResIdsFromList get redis pool error")
		return
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		resIdMidStrings, err = redis.Strings(conn.Do("LRANGE", key, 0, -1))
		if err != nil {
			continue
		}
		break
	}
	return
}

func ExistsResIdsList(key string) (exists bool, err error) {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		err = errors.New("ExistsResIdsList get redis pool error")
		return
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		exists, err = redis.Bool(conn.Do("EXISTS", key))
		if err != nil {
			continue
		}
		break
	}
	return
}

//func GetUserResUpdateRecord(key int64) (int64, error) {
//	pool := conf.GetRedisPool(XngUserDocUpdate)
//	if pool == nil {
//		return 0, errors.New("get redis pool error")
//	}
//	conn := pool.Get()
//	mid, err := redis.Int64(conn.Do("GET", key))
//	if err != nil {
//		return 0, err
//	}
//	return mid, nil
//}
//
//func AddUserResUpdateRecord(key int64, value int64) error {
//	pool := conf.GetRedisPool(XngUserDocUpdate)
//	if pool == nil {
//		return errors.New("AddUserResUpdateRecord get redis pool error")
//	}
//	conn := pool.Get()
//	exist, err := redis.Bool(conn.Do("EXISTS", key))
//	if err != nil {
//		//conf.Logger.Error("juedge user resource record exist error, resId=%v, mid=%v, error=%v", key, value, err)
//		return err
//	}
//	if exist == true {
//		return nil
//	}
//	_, err = conn.Do("SETEX", key, TransRecordLifeTime, value)
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func DelUserResUpdateRecord(key int64) error {
//	pool := conf.GetRedisPool(XngUserDocUpdate)
//	if pool == nil {
//		return errors.New("get redis pool error")
//	}
//	conn := pool.Get()
//	_, err := conn.Do("DEL", key)
//	if err != nil {
//		return err
//	}
//	return nil
//}

func AddResIdToList(key string, resId, mid int64) (err error) {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		return errors.New("get redis pool error")
	}
	val := fmt.Sprintf("%d:%d", resId, mid)

	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		_, err = conn.Do("LPUSH", key, val)
		if err != nil {
			continue
		}
		break
	}
	//exist, err := redis.Int(conn.Do("TTL", key))
	//if err != nil {
	//	//conf.Logger.Error("ttl life time error", "qid", key, "error: ", err)
	//	return err
	//}
	//if exist == LifeTimeNotSet {
	//	_, err = conn.Do("EXPIRE", key, TransRecordLifeTime) //时间放长，一天
	//	if err != nil {
	//		//conf.Logger.Error("set life time for key error", "qid: ", key, "error: ", err)
	//		return err
	//	}
	//}
	return
}

func DelUserResIdsList(key string) error {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		return errors.New("get redis pool error")
	}
	conn := pool.Get()
	_, err := conn.Do("DEL", key)
	if err != nil {
		return err
	}
	return nil
}

func AddPushMessage(items api.AlbumSuccessItems) error {
	pool := conf.GetRedisPool(XngAlbumRedis)
	if pool == nil {
		return errors.New("AddPushMessage get redis pool error")
	}
	conn := pool.Get()
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}

	msg := string(data)
	_, err = conn.Do("LPUSH", XngAlbumSuccessQueue, msg)
	if err != nil {
		return err
	}
	xlog.Info("success push message, msg:%s", msg)
	return nil
}

func AddPushMessageForFail(items api.AlbumFailItems) error {
	pool := conf.GetRedisPool(XngAlbumRedis)
	if pool == nil {
		return errors.New("get redis pool error")
	}
	conn := pool.Get()
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	_, err = conn.Do("LPUSH", XngAlbumFailQueue, string(data))
	if err != nil {
		return err
	}
	return nil
}

func AddNewResInfoForHandle(key string, doc *api.XngResourceInfoDoc) (err error) {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		return errors.New("AddNewResInfoForHandle get redis pool error")
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		_, err = conn.Do("SET", key, string(data))
		if err != nil {
			continue
		}
		break
	}
	return
}

func GetNewResInfoForHandle(key string) (doc *api.XngResourceInfoDoc, err error) {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		err = errors.New("GetNewResInfoForHandle get redis pool error")
		return
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		data, err := redis.Bytes(conn.Do("GET", key))
		if err != nil {
			continue
		}
		err = json.Unmarshal(data, &doc)
		if err != nil {
			continue
		}
		break
	}
	return
}

func DelNewResInfoForHandle(key string) (err error) {
	pool := conf.GetRedisPool(ResourcesRedis)
	if pool == nil {
		return errors.New("DelNewResInfoForHandle get redis pool error")
	}
	for i := 0; i < api.ReTryTimes; i++ {
		conn := pool.Get()
		_, err = conn.Do("DEL", key)
		if err != nil {
			continue
		}
		break
	}
	return
}
