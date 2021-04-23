package skel

import (
	"fmt"
	"testing"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
)

func TestDo(t *testing.T) {
	pool := conf.GetRedisPool("test")
	valStr, err := pool.GetString("test_key")
	if err != nil {
		return
	}
	fmt.Println(valStr)
}
