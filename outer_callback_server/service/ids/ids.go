package ids

import (
	"xgit.xiaoniangao.cn/xngo/service/ids_api"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/conf"
)

//GetMsgID 获取消息自增id
func GetMsgID() (id int64, err error) {
	newIdsRes, err := ids_api.GetNewIds(conf.C.Addrs.Ids, "xng-ids")

	if err != nil {
		return
	}
	return newIdsRes.Data["id"], nil
}
