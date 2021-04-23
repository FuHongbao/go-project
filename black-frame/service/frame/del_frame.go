package frame

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/util/common"
)

func DelBlackFrame(ctx context.Context, key string, startTime float64) (filePath string, err error) {
	/*
		path := common.GetSrcPath(key)
		destPath := common.GetDestPath(key)
		st := fmt.Sprintf("%f", startTime)
		cmd := exec.Command("cp", path, destPath)
		if err = cmd.Run(); err != nil {
			return
		}
		filePath = destPath
		return
	*/
	path := common.GetSrcPath(key)
	destPath := fmt.Sprintf("%s.mp4", common.GetDestPath(key))
	st := fmt.Sprintf("%f", startTime)
	cmd := exec.Command("ffmpeg", "-y", "-ss", st, "-v", "warning", "-i", path, "-c:v", "h264", destPath)
	//cmd := exec.Command("ffmpeg", "-v", "warning", "-ss", st, "-i", path, "-y", destPath)
	xlog.DebugC(ctx, "DelBlackFrame cmd:[%v]", cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		xlog.ErrorC(ctx, "DelBlackFrame.Command failed, err:[%s]", stderr.String())
		return
	}
	filePath = destPath
	return
}
func DelBlackFrameVideo(ctx context.Context, path string) (err error) {
	err = os.Remove(path)
	if err != nil {
		return
	}
	xlog.DebugC(ctx, "DelBlackFrameVideo del video:[%s]", path)
	return
}

/*
func DelBlackFrameVideo(ctx context.Context, key string) (err error) {
	path := common.GetSrcPath(key)
	destPath := common.GetDestPath(key) + ".mp4"
	err = os.Remove(path)
	if err != nil {
		return
	}
	xlog.DebugC(ctx, "DelBlackFrameVideo del video:[%s]", path)
	err = os.Remove(destPath)
	if err != nil {
		return
	}
	xlog.DebugC(ctx, "DelBlackFrameVideo del video:[%s]", destPath)
	return
}
*/
func DelBlackFrameSnap(ctx context.Context, key string) (err error) {
	destPath := common.GetDestPath(key) + ".jpg"
	err = os.Remove(destPath)
	if err != nil {
		return
	}
	xlog.DebugC(ctx, "DelBlackFrameSnap del snap:[%s]", destPath)
	return
}
