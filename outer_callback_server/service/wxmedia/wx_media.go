package wxmedia

import (
	"errors"
	"xgit.xiaoniangao.cn/xngo/service/outer_callback_server/dao/wxmediadao"
)

//Add 将下载到的数据插入素材库
func Add(msg *wxmediadao.WXMedia) (err error) {
	if msg == nil || msg.ID == 0 {
		return errors.New("wx media param err")
	}

	err = wxmediadao.Dao.Insert(msg)
	return
}
