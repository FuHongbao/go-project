package utils

import (
	"strings"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
)

const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

//OpenidToNum defined
func OpenidToNum(openid string) int64 {
	ret := int64(0)
	for i := 6; i < 14; i++ {
		idx := strings.Index(alphabet, string(openid[i]))
		ret = ret*64 + int64(idx)
	}
	return ret
}

//InStrArray 判断str是否在arr中
func InStrArray(arr []string, str string) bool {
	for _, v := range arr {
		if v == str {
			return true
		}
	}
	return false
}

//InterfaceToInt interface 转为int
func InterfaceToInt(value interface{}) (op int) {
	switch value.(type) {
	case int:
		op, _ = value.(int)
		return op
	default:
		xlog.Debug("interface trans int fail")
		return 0
	}
}

//InterfaceToString interface 转为string
func InterfaceToString(value interface{}) (op string) {
	switch value.(type) {
	case string:
		op, _ = value.(string)
		return op
	default:
		xlog.Debug("interface trans string fail")
		return ""
	}
}
