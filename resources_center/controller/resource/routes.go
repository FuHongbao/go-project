package resource

import (
	"github.com/gin-gonic/gin"
)

func InitRouters(app *gin.Engine) {

	group := app.Group("resource")
	group.POST("get_upload_status", GetUploadStatus)                //获取上传状态接口
	group.POST("get_upload_info", GetUploadInfo)                    //获取上传配置信息
	group.POST("static_upload_config", GetStaticResUploadConfig)    //获取static资源上传配置信息
	group.POST("multi_upload_config", GetMultiUploadConfig)         //获取分片配置信息，整合了续传和分片逻辑
	group.POST("check_multi_upload_result", CheckMultiUploadResult) //合并文件，校验上传结果
	group.POST("oss_callback", DealOssCallBack)
	group.POST("op_upload_conf", GetOpUploadInfo)             //浏览器上传配置接口
	group.POST("check_op_upload_result", CheckOpUploadResult) //浏览器上传信息校验接口
	group.POST("get_by_id", ByID)                             // 获取资源信息根据id
	group.POST("get_by_qetag", ByEtag)                        //根据etag获取视频信息

	group2 := app.Group("resource/v2")
	group2.POST("check_multi_upload_result", CheckMultiUploadResultV2) //合并文件，校验上传结果,V2版
	group2.POST("check_op_upload_result", CheckOpUploadResultV2)       //浏览器上传信息校验接口V2版(异步获取信息，支持更多视频格式)
	group2.POST("get_by_id", ByIDBatch)                                // 获取资源信息根据id
	group.POST("mediaInfo_callback", DealMediaInfoCallBack)
	group.POST("res_exists", CheckResExists)

	group.POST("set_m3u8_id", SetM3u8ExpID) //小程序m3u8实验
}
