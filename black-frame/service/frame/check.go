package frame

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
)

const (
	ScriptPath = "./algorithm/detectBlackFrame.py"
)

func CheckBlackFrame(ctx context.Context, path string) (ok bool, frameTime float64, err error) {
	args := []string{ScriptPath, "-i", path}
	out, err := exec.Command("python3", args...).Output()
	if err != nil {
		return
	}
	temp := string(out)
	result := strings.Replace(temp, "\n", "", -1)
	if result == "" {
		err = errors.New(fmt.Sprintf("CheckBlackFrame.Command failed, result is nil"))
		return
	}
	frameTime, err = strconv.ParseFloat(result, 64)
	if err != nil {
		return
	}
	if frameTime == -1 {
		err = errors.New(fmt.Sprintf("CheckBlackFrame.Command failed, The path:[%s] is wrong or the video is broken.", path))
		return
	} else if frameTime == 0 {
		ok = true
		return
	}
	xlog.DebugC(ctx, "python result:[%v], cmd:[python3 %v], out:[%s]", frameTime, args, temp)
	/*
		if result == "-1" {
			err = errors.New(fmt.Sprintf("CheckBlackFrame.Command failed, The path:[%s] is wrong or the video is broken.", path))
			return
		} else if result == "0" {
			ok = true
			return
		}
		frameTime, err = strconv.ParseFloat(result, 64)
		if err != nil {
			return
		}
	*/
	return
}
