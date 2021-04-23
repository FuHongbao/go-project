package api

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/services/mts"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/service/sts"
)

const (
	CacheShortTime  = 5 * 60
	CacheMiddleTime = 15 * 60
)

// WatermarkOne ...
type WatermarkOne struct {
	Ty          int    `json:"ty"`
	Key         string `json:"key"`
	QS          string `json:"qs"`
	WatermarkQS string `json:"watermark_qs"`
}

const (
	TypeWatermarkImg  = 1
	TypeWatermarkText = 2
)

//获取上传状态请求结构体
type UploadStatusReq struct {
	QeTag string `json:"qetag" binding:"gt=8"`
	Type  int    `json:"ty"` //资源类型： 12：影集；6：视频；
}

//获取上传状态响应结构体
type UploadStatusResp struct {
	Status int `json:"exist"`
}

//设置上传状态请求结构体
type SetUploadStatusReq struct {
	QeTag  string `json:"qetag"`
	Qid    int64  `json:"qid"`
	Status int    `json:"status"`
}

//客户端上传成功后的回调
type AppUploadCallbackReq struct {
	QeTag    string `json:"qetag" binding:"required"` //生成方式参考: https://github.com/qiniu/qetag
	Qid      int64  `json:"qid" binding:"required"`
	Mid      int64  `json:"mid" binding:"required"`
	FileType int    `json:"type" binding:"required"` //6：视频 1：图片，目前仅支持视频
}

//临时上传凭证请求结构体, 已废弃
type TempVoucherReq struct {
	Token string `json:"token"`
}

//临时上传凭证响应结构体
type TempVoucherResp struct {
	Endpoint      string `json:"endpoint"`
	EndpointInter string `json:"endpoint_internal"`
	Bucket        string `json:"bucket"`
	AccessKey     string `json:"accessKey"`
	SecretKey     string `json:"secretKey"`
	SecurityToken string `json:"securityToken"`
	Region        string `json:"region"`
	Qid           int64  `json:"qid"`
}

//文件信息请求结构体
type MediaInfoReq struct {
	Mid int64 `json:"mid"`
	Qid int64 `json:"qid"` //FIXME：这个参数是qid还是用户资源id呢？ 如果是qid参数名字最好写qid吧
	//Key   string `json:"key"`
	QeTag string `json:"qetag"` //todo::驼峰
}

//文件信息响应结构体
type MediaInfoResp struct {
	Id       int64   `json:"id"`
	Qid      int64   `json:"qid"`
	Ty       int     `json:"ty"`
	Size     int64   `json:"size"`
	VideoUrl string  `json:"v_url"`
	Url      string  `json:"url"`
	Upt      int64   `json:"upt"`
	Mt       int64   `json:"mt"`
	Ct       int64   `json:"ct"`
	Src      string  `json:"src"`
	Fmt      string  `json:"fmt"`
	W        int     `json:"w"`
	H        int     `json:"h"`
	Du       float64 `json:"du"`
	Cover    int64   `json:"cover"`
	Code     string  `json:"code"`
	QeTag    string  `json:"qetag"`
}

//提交转码请求结构体
type TransCodeReq struct {
	Qid int64 `json:"qid"`
	//Key string `json:"key"`
}

//提交转码响应结构体
type TransCodeResp struct {
	Status int `json:"status"`
}

//处理转码状态请求结构体
type CheckStatusReq struct {
	Qid   int64 `json:"qid"`
	Aid   int64 `json:"aid"`
	ResId int64 `json:"id"`
}

//资源库qetag集合文档
type XngTagInfoDoc struct {
	QeTag string `json:"_id" bson:"_id"`
	Qid   int64  `json:"qid" bson:"qid"`
}

//资源库qid集合文档
type XngResourceInfoDoc struct {
	ResId     int64      `json:"_id" bson:"_id"`
	Type      int        `json:"ty" bson:"ty,omitempty"`
	Size      int64      `json:"size" bson:"size,omitempty"`
	QeTag     string     `json:"qetag" bson:"qetag,omitempty"`
	Upt       int64      `json:"upt" bson:"upt,omitempty"`
	Mt        int64      `json:"mt" bson:"mt,omitempty"`
	Ct        int64      `json:"ct,omitempty" bson:"ct,omitempty"`
	Src       string     `json:"src" bson:"src,omitempty"`
	Fmt       string     `json:"fmt" bson:"fmt,omitempty"`
	Ort       int        `json:"ort" bson:"ort,omitempty"`
	W         int        `json:"w" bson:"w,omitempty"`
	H         int        `json:"h" bson:"h,omitempty"`
	Ref       int        `json:"ref" bson:"ref,omitempty"`
	Du        float64    `json:"du,omitempty" bson:"du,omitempty"`
	Cover     int64      `json:"cover,omitempty" bson:"cover,omitempty"`
	CoverTp   string     `json:"covertp,omitempty" bson:"covertp,omitempty"`
	Code      string     `json:"code,omitempty" bson:"code,omitempty"`
	TransCode *int       `json:"trans,omitempty" bson:"trans"`
	MusicName string     `json:"music_name,omitempty"` //音乐类资源的歌曲名字
	VRate     []RateInfo `json:"vrate,omitempty"`      //资源对应的转码信息
}
type RateInfo struct {
	TplID string `json:"tpl_id"` //码率模板id，标识一个转码文件的分辨率，宽高设置，视频封装格式，编码格式等
	ID    string `json:"id"`     //转码资源的id
}

//用户资源库文档
type UserResourceDoc struct {
	ResId int64   `json:"_id" bson:"_id"` //FIXME:json是json格式解析定义的，一般用来接收http请求解析json串定义这个tag， mongodb是bson 需要定义bson格式
	Size  int64   `json:"size" bson:"size,omitempty"`
	QeTag string  `json:"qetag" bson:"qetag,omitempty"`
	Upt   int64   `json:"upt" bson:"upt,omitempty"`
	Mt    int64   `json:"mt" bson:"mt,omitempty"`
	Ct    int64   `json:"ct" bson:"ct,omitempty"`
	Src   string  `json:"src" bson:"src,omitempty"`
	Fmt   string  `json:"fmt" bson:"fmt,omitempty"`
	Ort   int     `json:"ort" bson:"ort,omitempty"`
	W     int     `json:"w" bson:"w,omitempty"`
	H     int     `json:"h" bson:"h,omitempty"`
	Qid   int64   `json:"qid" bson:"qid,omitempty"`
	Mid   int64   `json:"mid" bson:"mid,omitempty"`
	Du    float64 `json:"du" bson:"du,omitempty"`
	Cover int64   `json:"cover" bson:"cover,omitempty"`
	Ty    int     `json:"ty" bson:"ty,omitempty"`
	Code  string  `json:"code" bson:"code,omitempty"`
	Dt    int     `json:"dt" bson:"dt,omitempty"`
	D     int     `json:"d" bson:"d,omitempty"`
}

type MNSMessageData struct {
	JobID    string `json:"jobId"`
	Type     string `json:"type"`
	State    string `json:"state"`
	UserData string `json:"userData"`
}

//消息服务结构体
type MNSMessage struct {
	TopicOwner         string `xml:"TopicOwner"`
	TopicName          string `xml:"TopicName"`
	Subscriber         string `xml:"Subscriber"`
	Message            []byte `xml:"Message"`
	MessageMD5         string `xml:"MessageMD5"`
	MessageId          string `xml:"MessageId"`
	MessagePublishTime string `xml:"MessagePublishTime"`
}

//后期资源doc的id将变更为string类型,暂时与大文件区分开,只对外展示使用,不存入数据库
type ResourceInfoDoc struct {
	ResId     string  `json:"id" bson:"id"`
	Type      int     `json:"ty" bson:"ty,omitempty"`
	Size      int64   `json:"size" bson:"size,omitempty"`
	QeTag     string  `json:"qetag" bson:"qetag,omitempty"`
	Upt       int64   `json:"upt" bson:"upt,omitempty"`
	Mt        int64   `json:"mt" bson:"mt,omitempty"`
	Ct        int64   `json:"ct" bson:"ct,omitempty"`
	Src       string  `json:"src" bson:"src,omitempty"`
	Fmt       string  `json:"fmt" bson:"fmt,omitempty"`
	Ort       int     `json:"ort" bson:"ort,omitempty"`
	W         int     `json:"w" bson:"w,omitempty"`
	H         int     `json:"h" bson:"h,omitempty"`
	Ref       int     `json:"ref" bson:"ref,omitempty"`
	Du        float64 `json:"du" bson:"du,omitempty"`
	Cover     int64   `json:"cover" bson:"cover,omitempty"`
	CoverTp   string  `json:"covertp" bson:"covertp,omitempty"`
	Code      string  `json:"code" bson:"code,omitempty"`
	TransCode *int    `json:"trans" bson:"trans"`
}

type GetUploadConfReq struct {
	Kind    int    `json:"type"`
	QeTag   string `json:"qetag"`
	Product int    `json:"prod"` //产品 1 xng 2 xbd 3 tia
	Project string `json:"proj"` //ma app ...
}

type CallbackCustomParam struct {
	Kind      int    `json:"type"`
	QeTag     string `json:"qetag"`
	Product   int    `json:"prod"` //产品 1 xng 2 xbd 3 tia
	Project   string `json:"proj"` //ma app ...
	NoMq      int    `json:"no_mq"`
	MusicName string `json:"music_name,omitempty"` //音乐类资源的歌曲名字
}

type UploadInfo struct {
	Signature           string `json:"Signature"`
	Host                string `json:"-"`
	InternalHost        string `json:"-"`
	Policy              string `json:"policy"`
	Callback            string `json:"Callback"`
	Key                 string `json:"key"` //字段变为id，后期类型将变为string类型
	AccessKey           string `json:"OSSAccessKeyId"`
	SuccessActionStatus string `json:"success_action_status"`
	SecurityToken       string `json:"x-oss-security-token"`
	ExpireSec           int    `json:"-"`
}

type UploadConfCommonResp struct {
	Signature           string              `json:"signature"`
	Host                string              `json:"host"`
	Policy              string              `json:"policy"`
	Callback            string              `json:"callback"`
	Qid                 string              `json:"id"` //字段变为id，后期类型将变为string类型
	Dir                 string              `json:"dir,omitempty"`
	AccessKey           string              `json:"access_key"`
	SuccessActionStatus string              `json:"success_action_status"`
	CustomData          CallbackCustomParam `json:"custom_data"` //前端将该自定义数据，一块请求oss。形式{"x:field" :value}
	//ExpireSec int64  `json:"expire_sec"`
	//SecretKey     string `json:"secret_key"`
	SecurityToken string `json:"security_token"`
	//Endpoint         string `json:"endpoint"`
	//EndpointInternal string `json:"endpoint_internal"`
	//RequestId        string `json:"request_id"`
	//Bucket           string `json:"bucket"`
}

//type DealOssCallBackReq struct {
//	Bucket   string `json:"bucket" binding:"required"`
//	Filename string `json:"filename" binding:"required"` //文件名即资源qid,后期更新为string类型
//	Size     int64  `json:"size"`
//	MimeType string `json:"mimeType"`
//	Height   int    `json:"height"`
//	Width    int    `json:"width"`
//	Format   string `json:"format"`
//	MyVar    string `json:"my_var" binding:"required"`
//	//GetUploadConfReq
//}

type ReqStaticUploadConf struct {
	FileName    string `json:"filename"`
	YWSide      string `json:"yw_side"`
	Path        string `json:"path"`
	Prod        int    `json:"prod"`
	ContentType string `json:"content_type"`
}

type RespStaticUploadConf struct {
	Method        string `json:"method"`
	Url           string `json:"url"`
	UrlInternal   string `json:"url_internal"`
	Host          string `json:"host"`
	HostInternal  string `json:"host_internal"`
	Date          string `json:"date"`
	SecurityToken string `json:"security_token"`
	ExpireSec     int    `json:"expire_sec"`
	Authorization string `json:"Authorization"`
}

type DealOssCallBackResp struct {
	Status string `json:"Status"`
	ResourceInfoDoc
}

type AlbumSuccessItems struct {
	ID       int64   `json:"album_id"`
	TryTimes int     `json:"trys"`
	Size     int64   `json:"size"`
	Du       float64 `json:"du"`
	VW       int     `json:"vw"`
	VH       int     `json:"vh"`
	//Uflag  int      `json:"uflag"`    影集更换音乐
}

type AlbumFailItems struct {
	ID    int64 `json:"id"`
	Errno int   `json:"errno"`
}

type ReqGetMultiUploadConf struct {
	Kind  int    `json:"type"`
	QeTag string `json:"qetag"`
	Size  int64  `json:"size"`
}
type ChunkData struct {
	Number int   `json:"chunk_num"`
	Offset int64 `json:"offset"`
	Size   int64 `json:"chunk_size"`
	Ready  bool  `json:"ready"`
}
type PartData struct {
	Etag string `json:"etag"`
	Num  int    `json:"num"`
}
type RespMultiUploadConf struct {
	Key      string      `json:"key"`
	UploadID string      `json:"upload_id"`
	Chunks   []ChunkData `json:"chunks"`
	ChunkCnt int         `json:"chunk_cnt"`
	Parts    []PartData  `json:"parts"`
}
type MultiUploadRecord struct {
	Key      string `json:"key"`
	UploadID string `json:"upload_id"`
	Size     int64  `json:"size"`
}

type MultiAbortRecord struct {
	Key      string `json:"key"`
	UploadID string `json:"upload_id"`
	QeTag    string `json:"qetag"`
}

type MultiMediaInfo struct {
	W    int     `json:"w"`
	H    int     `json:"h"`
	Du   float64 `json:"du"`
	Size int64   `json:"size"`
	Code string  `json:"code"`
	Fmt  string  `json:"fmt"`
}

type ReqCheckMultiUpload struct {
	Kind        int        `json:"type"`
	QeTag       string     `json:"qetag"`
	Product     int        `json:"prod"` //产品 1 xng 2 xbd 3 tia
	Project     string     `json:"proj"` //ma app ...
	MediaInfo   string     `json:"media_info"`
	UserData    string     `json:"user_data"`
	Key         string     `json:"key"`
	UploadID    string     `json:"upload_id"`
	ContentType string     `json:"Content-Type"`
	Parts       []PartData `json:"parts"`
}

type ResourceInfo struct {
	ResId string  `json:"id"`
	Type  int     `json:"ty"`
	Size  int64   `json:"size"`
	QeTag string  `json:"qetag"`
	Upt   int64   `json:"upt"`
	Fmt   string  `json:"fmt"`
	W     int     `json:"w"`
	H     int     `json:"h"`
	Du    float64 `json:"du,omitempty"`
	Cover string  `json:"cover,omitempty"`
	Code  string  `json:"code,omitempty"`
	Ort   int     `json:"ort,omitempty"`
}
type RespCheckMultiUpload struct {
	Status int          `json:"status"`
	Info   ResourceInfo `json:"info"`
}

type ReqMultiPartAuth struct {
	Key      string `json:"key"`
	UploadID string `json:"upload_id"`
	ChunkNum int    `json:"chunk_num"`
	Md5Value string `json:"md5_value"`
	Kind     int    `json:"type"`
}

type RespMultiPartAuth struct {
	Url           string `json:"url"`
	Authorization string `json:"authorization"`
	Date          string `json:"date"`
	Token         string `json:"x-oss-security-token"`
	Host          string `json:"host"`
	Method        string `json:"method"`
	ContentType   string `json:"Content-Type"`
	ExpireSec     int    `json:"expire_sec"`
}

type MultiVideoUserData struct {
	UserData    string `json:"user_data"`
	Qetag       string `json:"qetag"`
	UserService string `json:"service_name"`
	OldId       string `json:"old_id,omitempty"`
	TplId       string `json:"tpl_id,omitempty"`
}

type ResDocWithCoverUrl struct {
	ResId            string  `json:"id"`
	Type             int     `json:"ty"`
	Size             int64   `json:"size"`
	QeTag            string  `json:"qetag"`
	Upt              int64   `json:"upt"`
	Fmt              string  `json:"fmt"`
	W                int     `json:"w"`
	H                int     `json:"h"`
	Du               float64 `json:"du,omitempty"`
	Cover            string  `json:"cover,omitempty"`
	Code             string  `json:"code,omitempty"`
	Ort              int     `json:"ort,omitempty"`
	CoverUrl         string  `json:"cover_url,omitempty"`
	CoverUrlInternal string  `json:"cover_url_internal,omitempty"`
	JobId            string  `json:"jobID,omitempty"`
	OldId            string  `json:"old_id,omitempty"`
	UserData         string  `json:"user_data,omitempty"`
}

const (
	XngStsVoucherUrlForTest = "https://uptokssl.xiaoniangao.cn/ali/test_frontend_upload_uptoken" //获取临时凭证接口路径 (用户使用)
	XngStsVoucherUrlForProd = "http://uptokssl-internal.xiaoniangao.cn/ali/frontend_upload_uptoken"

	XngStsForMtsUrlForTest = "https://uptokssl.xiaoniangao.cn/ali/test_upload_to_mts_uptoken" //临时凭证接口路径 （后端使用）
	XngStsForMtsUrlForProd = "http://uptokssl-internal.xiaoniangao.cn/ali/upload_to_mts_uptoken"

	ResourceUnKonwType  = -1   //未知类型资源对应值，暂未使用
	UploadFromWXMiniApp = "11" //资源来源：微信小程序
	UploadFromApp       = "13"
	ContentTypeVideo    = "video/mp4"
	ContentTypeImg      = "image/jpeg"
	ContentTypeMusic    = "audio/mpeg"
	ContentTypeTxt      = "text/html"
	ContentTypeLrc      = "text/html"
	ContentTypeJson     = "application/json"
	CopyResCutSize      = 268435456 //资源拷贝分片大小, 每片256m
	TransLimitSize      = 41943040  //转码大小限制，低于40m不转码

	SnapShotType = "jpeg" //截图类型，暂时固定为jepg
	//VideoDurationForSnapShot = 5000                               //用作截图时，若视频长度大于5秒，则从第五秒截一张图，否则截第一秒
	//SnapConfNum              = 1                                  //截图数量，暂时只截一张
	SnapConfTime      = 5000 //截图起始时间，时长超过5秒，截取第五秒
	SnapConfShortTime = 1000 //截图起始时间，时长不足5秒，截取第一秒
	//SnapConfInterval         = 0                                  //截图间隔：对于截取多张的情景，此参数代表间隔多少秒截取一次
	//SnapPipeLineID           = "f4b08ef3b60e4f01a71a46c7f7957469" //截图管道ID

	TransCodeTemplateId    = "d458bbf57b16ffd93237de548d5269db" //转码模板ID
	TransCodePipelineId    = "22f088f8bfef431a89887ae0bcefa55e" //转码管道ID  （线上环境）
	TransCodePipeIdForTest = "f4b08ef3b60e4f01a71a46c7f7957469" //转码管道  （测试环境）
	MediaInfoPipeLineId    = "680188f7d7fc44a2b5fb98d0b99e8e83"
	MergeVideoPipeLineId   = "680188f7d7fc44a2b5fb98d0b99e8e83" //合并视频管道 （线上与测试环境使用同一个管道）

	NotifyStatusSuccess = "Success"   //消息回调状态（成功）
	NotifyTypeTrans     = "Transcode" //消息回调类型（转码）
	PushMessageTrys     = 0

	MtsVpcDomain = "mts-vpc.cn-shenzhen.aliyuncs.com"

	//UploadCallBackUrl = "http://test-kapi.xiaoniangao.cn/resources_center/resource/oss_callback" //测试环境的回调url,已移动到配置文件
	//UploadCallBackUrl           = ""                                                                    //正式环境的回调url
	//UploadImgCallBackBody       = "bucket=${bucket}&filename=${object}&size=${size}&mimeType=${mimeType}&height=${imageInfo.height}&width=${imageInfo.width}&format=${imageInfo.format}&qetag=${x:qetag}&proj=${x:proj}&prod=${x:prod}&type=${x:type}&my_var=${x:my_var}"
	UploadImgCallBackBody = "{\"bucket\":${bucket},\"filename\":${object},\"size\":${size},\"mimeType\":${mimeType},\"height\":${imageInfo.height},\"width\":${imageInfo.width},\"format\":${imageInfo.format},\"my_var\":${x:my_var}}"
	//UploadResCommonCallBackBody = "bucket=${bucket}&filename=${object}&size=${size}&mimeType=${mimeType}&qetag=${x:qetag}&proj=${x:proj}&prod=${x:prod}&type=${x:type}" //设置视频资源上传后回调的内容
	UploadResCommonCallBackBody = "{\"bucket\":${bucket},\"filename\":${object},\"size\":${size},\"mimeType\":${mimeType},\"my_var\":${x:my_var}}" //设置视频资源上传后回调的内容
	UploadDir                   = ""

	OssSuccStatus = "OK"
)

// resource type from php
const (
	ResourceTypeImg        = 1 //图片类型资源对应值
	ResourceTypeVoice      = 5
	ResourceTypeVideo      = 6 //视频类型资源对应值
	ResourceTypeMusic      = 7
	ResourceTypeAvatar     = 8
	ResourceTypeGroupImg   = 9
	ResourceTypeLyric      = 10 //lrc歌词类型资源对应值
	ResourceTypeTxt        = 11 //txt格式资源对应值
	ResourceTypeAlbum      = 12 //影集资源类型对应值
	ResourceTypeLive       = 13 //直播类型资源对应值
	ResourceTypeGuideVideo = 14 //导播视频类型资源对应值
	ResourceTypeAPPLog     = 15 //app日志类型资源对应值
)

//产品类型
const (
	ResProductXng    = 1 //小年糕
	ResProductXbd    = 2 //小板凳
	ResProductTia    = 3 //TIA
	ResProductScreen = 4 //轻剪
)

var MqTopicMap = map[int]string{
	ResProductXng: "topic_upload_xng",
	ResProductXbd: "topic_upload_xbd",
	ResProductTia: "topic_upload_tia",
}
var MqMergeTopicMap = map[int]string{
	ResProductXng: "topic_merge_xng",
	ResProductXbd: "topic_merge_xbd",
	ResProductTia: "topic_merge_tia",
}
var MqTransTopicMap = map[int]string{
	ResProductXng: "topic_trans_xng",
	ResProductXbd: "topic_trans_xbd",
	ResProductTia: "topic_trans_tia",
}
var MqInfoTopicMap = map[int]string{
	ResProductXng:    "topic_info_xng",
	ResProductXbd:    "topic_info_xbd",
	ResProductTia:    "topic_info_tia",
	ResProductScreen: "topic_info_screen",
}
var MqMergeTagMap = map[int]string{
	ResourceTypeImg:      "img",
	ResourceTypeVoice:    "music",
	ResourceTypeVideo:    "video",
	ResourceTypeMusic:    "music",
	ResourceTypeGroupImg: "img",
	ResourceTypeLyric:    "text",
	ResourceTypeTxt:      "text",
	ResourceTypeAlbum:    "album",
	ResourceTypeLive:     "live",
}
var MqTransTagMap = MqMergeTagMap

//var MqInfoTagMap = MqMergeTagMap
var StaticUploadMap = map[int]string{
	ResProductXng: "xng",
	ResProductXbd: "xbd",
	ResProductTia: "tia",
}

const (
	StsForUserUpload      = "common_upload_user"
	StsForMtsMedia        = "common_upload_mts"
	StsForStaticUpload    = "common_upload_static"
	StsForAlbumUpload     = "album"
	StsForMtsAlbum        = "common_album_mts"
	StsForMtsLive         = "common_live_mts"
	StsForLiveGuideUpload = "live_guide_user"
	StsForMtsLiveGuide    = "live_guide_mts"
	StsForXNGAppLog       = "app_log"
)

const (
	ReTryTimes = 3
)
const (
	MultiPartContentType = "application/octet-stream"
)

const (
	CronMachineCnt        = 3
	CronCleanTaskMachine1 = 1
	CronCleanTaskMachine2 = 2
	CronCleanTaskMachine3 = 3
)

var CleanTaskMap = map[int]string{
	CronCleanTaskMachine1: "ecs-hn1e-resources-center-1",
	CronCleanTaskMachine2: "ecs-hn1e-resources-center-2",
	CronCleanTaskMachine3: "ecs-hn1d-resources-center-3",
}

const (
	MTSJobTypeMergeVideo           = "mergeVideo"
	MTSJobTypeTransVideoForBigFile = "transBigVideo"
	MTSJobTypeTransVideo           = "transVideo"
	MTSJobTypeVideoInfo            = "videoInfo"   //分片上传获取信息-标识
	MTSJobTypeOpVideoInfo          = "opVideoInfo" //op上传获取信息-标识
)

type AliOssClient struct {
	Client *oss.Client      `json:"client"`
	Sts    *sts.UploadToken `json:"sts"`
}
type AliMtsClient struct {
	Client *mts.Client      `json:"client"`
	Sts    *sts.UploadToken `json:"sts"`
}

var StsOssClient = map[string]*AliOssClient{}
var StsMtsClient = map[string]*AliMtsClient{}

const (
	TransTemplate720PMP4 = "000001"
)
