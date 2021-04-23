package frame

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/api"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/dao/mq"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/service/frame"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/service/res_url"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/service/upload"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/util/common"
)

type ReqDelBlackFrame struct {
	Key     string `json:"key"`
	Product int    `json:"prod"`
	Project string `json:"proj"`
}
type RespDelBlackFrame struct {
	Status   int    `json:"status"`
	ID       string `json:"id"`
	Cover    string `json:"cover"`
	CoverUrl string `json:"cover_url"`
}

func DelBlackFrame(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req ReqDelBlackFrame
	if !xc.GetReqObject(&req) {
		return
	}
	if req.Key == "" || req.Project == "" || req.Product <= 0 {
		xc.ReplyFail(lib.CodePara)
		return
	}
	//下载资源到本地src_video文件夹
	url, err := res_url.GetVideoUrlByID(xc, req.Key)
	if err != nil {
		xlog.ErrorC(xc, "DelBlackFrame.GetVideoUrlByID failed, err:[%v], req:[%v]", err, req)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	startTime := time.Now()
	path, err := res_url.DownLoadResource(xc, url, req.Key)
	if err != nil {
		xlog.ErrorC(xc, "DelBlackFrame.DownLoadResource failed, err:[%v], req:[%v], url:[%s]", err, req, url)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xlog.DebugC(xc, "DownLoadResource use time:[%v]", time.Since(startTime))
	//检验视频黑帧情况
	ok, frameTime, err := frame.CheckBlackFrame(xc, path)
	if err != nil {
		xlog.ErrorC(xc, "DelBlackFrame.CheckBlackFrame failed, err:[%v], req:[%v]", err, req)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	if ok == true {
		path := common.GetSrcPath(req.Key)
		err = frame.DelBlackFrameVideo(xc, path)
		if err != nil {
			xlog.ErrorC(xc, "DelBlackFrame.DelBlackFrameVideo failed, err:[%v], req:[%v]", err, req)
			return
		}
		xc.ReplyOK(RespDelBlackFrame{Status: 0})
		return
	}
	//获取资源新id
	var resKey, snapKey, snapUrl string
	ch := make(chan int64, 2)
	go upload.GetDistributedId(ch)
	go upload.GetDistributedId(ch)
	resId := <-ch
	if resId <= 0 {
		xlog.ErrorC(xc, "DelBlackFrame.GetDistributedId failed, err:[%v], req:[%v]", err, req)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	resKey = strconv.FormatInt(resId, 10)
	snapId := <-ch
	if snapId <= 0 {
		xlog.ErrorC(xc, "DelBlackFrame.GetDistributedId failed, err:[%v], req:[%v]", err, req)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	snapKey = strconv.FormatInt(snapId, 10)
	//截图并上传
	snapPath, err := frame.GetVideoSnap(xc, req.Key, frameTime, snapKey)
	if err != nil {
		xlog.ErrorC(xc, "DelBlackFrame.GetVideoSnap failed, err:[%v], req:[%v]", err, req)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	snapCh := make(chan string, 1)
	go func() {
		//获取截图链接url
		imgUrl, _, err := res_url.GetImgUrlByID(xc, snapKey)
		if err != nil {
			xlog.ErrorC(xc, "DelBlackFrame.GetImgUrlByID failed, err:[%v], req:[%v]", err, req)
			snapCh <- ""
			return
		}
		snapCh <- imgUrl
	}()
	snapUrl = <-snapCh
	if snapUrl == "" {
		xlog.ErrorC(xc, "DelBlackFrame.GetImgUrlByID failed, err:url is nil, req:[%v]", req)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = upload.ResUploadByResourceCenter(xc, snapPath, req.Product, req.Project, snapKey, 1)
	if err != nil {
		xlog.ErrorC(xc, "DelBlackFrame.ResUploadByResourceCenter for snap failed, err:[%v], req:[%v]", err, req)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	message := api.BlackFrameMqMessage{
		OldKey:    req.Key,
		NewKey:    resKey,
		SnapKey:   snapKey,
		Prod:      req.Product,
		Proj:      req.Project,
		FrameTime: frameTime,
		Url:       url,
	}
	err = mq.NotifyByMq(xc, api.TopicBlackFrame, "", message)
	if err != nil {
		xlog.ErrorC(xc, "DelBlackFrame.NotifyByMq failed, err:[%v], req:[%v]", err, req)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	err = frame.DelBlackFrameVideo(xc, path)
	if err != nil {
		xlog.ErrorC(xc, "DelBlackFrame.DelBlackFrameVideo failed, err:[%v], req:[%v]", err, req)
	}
	err = frame.DelBlackFrameSnap(xc, snapKey)
	if err != nil {
		xlog.ErrorC(xc, "DelBlackFrame.DelBlackFrameVideo failed, err:[%v], req:[%v]", err, req)
	}
	resp := RespDelBlackFrame{
		Status:   1,
		ID:       resKey,
		Cover:    snapKey,
		CoverUrl: snapUrl,
	}
	xc.ReplyOK(resp)
	//go func() {
	//	//ffmpeg工具生成新视频
	//	filePath, err := frame.DelBlackFrame(xc, req.Key, frameTime)
	//	if err != nil {
	//		xlog.ErrorC(xc, "DelBlackFrame.DelBlackFrame failed, err:[%v], req:[%v]", err, req)
	//		return
	//	}
	//	//上传新视频
	//	err = upload.ResUploadByResourceCenter(xc, filePath, req.Product, req.Project, resKey, 6)
	//	if err != nil {
	//		xlog.ErrorC(xc, "DelBlackFrame.ResUploadByResourceCenter for video failed, err:[%v], req:[%v]", err, req)
	//		return
	//	}
	//	//更新资源doc内的cover字段
	//	ret, err := res_info.UpdateResCoverByResCenter(xc, resKey, snapKey)
	//	if err != nil {
	//		xlog.ErrorC(xc, "DelBlackFrame.UpdateResCoverByResCenter failed, err:[%v], req:[%v]", err, req)
	//		return
	//	}
	//	if ret == 0 {
	//		xlog.ErrorC(xc, "DelBlackFrame.UpdateResCoverByResCenter failed, ret=0, resource may be not exist")
	//		return
	//	}
	//	//删除旧资源
	//	path := common.GetSrcPath(req.Key)
	//	err = frame.DelBlackFrameVideo(xc, path)
	//	if err != nil {
	//		xlog.ErrorC(xc, "DelBlackFrame.DelBlackFrameVideo failed, err:[%v], req:[%v]", err, req)
	//		return
	//	}
	//	destPath := common.GetDestPath(req.Key) + ".mp4"
	//	err = frame.DelBlackFrameVideo(xc, destPath)
	//	if err != nil {
	//		xlog.ErrorC(xc, "DelBlackFrame.DelBlackFrameVideo failed, err:[%v], req:[%v]", err, req)
	//		return
	//	}
	//	err = frame.DelBlackFrameSnap(xc, snapKey)
	//	if err != nil {
	//		xlog.ErrorC(xc, "DelBlackFrame.DelBlackFrameVideo failed, err:[%v], req:[%v]", err, req)
	//		return
	//	}
	//}()

}
