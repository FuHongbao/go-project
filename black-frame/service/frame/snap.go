package frame

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/util/common"
)

func GetVideoSnap(ctx context.Context, fileName string, startTime float64, snapName string) (snapUrl string, err error) {
	//snapUrl = "./dest_video/3785497.jpg"
	//return
	path := common.GetSrcPath(fileName)
	destPath := common.GetDestPath(snapName) + ".jpg"
	st := fmt.Sprintf("%f", startTime+1)
	cmd := exec.Command("ffmpeg", "-ss", st, "-i", path, "-vframes", "1", "-y", destPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	xlog.DebugC(ctx, "GetVideoSnap cmd:[%v]", cmd)
	if err = cmd.Run(); err != nil {
		xlog.ErrorC(ctx, "GetVideoSnap.Command failed err:[%s]", stderr.String())
		return
	}
	snapUrl = destPath
	return
}
