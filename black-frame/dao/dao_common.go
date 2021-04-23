package dao

import (
	"strings"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
)

const (
	TotalLogicDb       = 4096
	TotalDBNumber      = 4
	DbXngResource      = "xng_qiniu"
	ColCommonFileByQid = "qiniu_common_file_by_qid"
	BASESTR            = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
)

func GetKeyStr(qetag string) string {
	return qetag[len(qetag)-8:]
}

func GetMediaId(strcode string) int64 {
	alphabet := BASESTR
	var mediaId int64
	slen := len(strcode)
	for i := 0; i < slen; i++ {
		mediaId = mediaId*64 + int64(strings.Index(alphabet, string(strcode[i])))
	}
	return mediaId
}

func GetResMod(id int64) int {
	return int(id % TotalLogicDb)
}

func GetNodeMod(resMod int, dbName string) int {
	if dbName == "" {
		xlog.Error("get node mod error, dbName is nil")
		return -1
	}
	nodeMod := resMod % TotalDBNumber
	return nodeMod
}
