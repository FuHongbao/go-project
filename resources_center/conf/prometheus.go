package conf

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	MergeCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "merge_video_total",
			Help: "merge/video接口请求数",
		},
	)
	UrlImgCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_img_total",
			Help: "url/img接口请求数",
		},
	)
	UrlImgV2Counter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "url_img_v2_total",
			Help: "url/v2/img接口请求数",
		},
	)
	OssImgShotWithProcessCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "oss_imgShot_process_total",
		Help: "调用阿里云oss截帧持久化总次数",
	},
		[]string{"function"})
)
