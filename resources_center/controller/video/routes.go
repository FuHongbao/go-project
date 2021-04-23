package video

import (
	"github.com/gin-gonic/gin"
)

func InitRouters(app *gin.Engine) {
	group := app.Group("video")
	group.POST("get_upload_status", GetUploadStatus)           //获取上传状态接口
	group.POST("set_upload_status", SetUploadStatus)           //设置资源上传状态接口
	group.POST("sts_voucher", GetTempVoucher)                  //获取临时凭证接口
	group.POST("videoinfo", GetVideoInfo)                      //视频信息获取接口
	group.POST("transvideo", SubmitVideoTrans)                 //提交转码接口  (视频资源)
	group.POST("handle_trans_completed", HandleTransCompleted) //处理转码完成接口
	group.POST("callback", ResultCallBack)                     //消息回调接口
	group.POST("app_upload_callback", NewAppUploadCallback)    //客户端文件上传成功后回调该接口
	group.POST("resource_info_by_etag", GetResourceInfoByEtag) //根据etag获取视频信息

}
