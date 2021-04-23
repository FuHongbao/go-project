package queue

import (
	"context"
	"encoding/json"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/api"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/conf"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/service/frame"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/service/res_info"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/service/res_url"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/service/upload"
	"xgit.xiaoniangao.cn/xngo/service/black-frame/util/common"
)

func SubScribe(topic string, tag string) (err error) {
	selector := consumer.MessageSelector{}
	if tag != "" {
		selector.Type = consumer.TAG
		selector.Expression = tag
	}
	err = conf.MqConsumer.Subscribe(topic, selector, DealMessage)
	if err != nil {
		return
	}
	err = conf.MqConsumer.Start()
	if err != nil {
		xlog.ErrorC(context.Background(), "start consumer mq failed, err:[%v]", err)
		return
	}
	xlog.Debug("MqConsumer start SubScribe success, topic:[%s]", topic)
	return
}

func DealMessage(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		message := api.BlackFrameMqMessage{}
		err := json.Unmarshal(msg.Body, &message)
		if err != nil {
			xlog.ErrorC(ctx, "DealMessage.Unmarshal failed, msg:[%v], err:[%v]", msg.Body, err)
			continue
		}
		xlog.DebugC(ctx, "success get message:[%v]", message)
		//下载黑帧视频
		_, err = res_url.DownLoadResource(ctx, message.Url, message.OldKey)
		if err != nil {
			xlog.ErrorC(ctx, "DealMessage.DownLoadResource failed, err:[%v], req:[%v], url:[%s]", err, msg.Body, message.Url)
			continue
		}
		//ffmpeg工具生成新视频
		filePath, err := frame.DelBlackFrame(ctx, message.OldKey, message.FrameTime)
		if err != nil {
			xlog.ErrorC(ctx, "DealMessage.DelBlackFrame failed, err:[%v], msg:[%v]", err, message)
			continue
		}
		//上传新视频
		err = upload.ResUploadByResourceCenter(ctx, filePath, message.Prod, message.Proj, message.NewKey, 6)
		if err != nil {
			xlog.ErrorC(ctx, "DealMessage.ResUploadByResourceCenter for video failed, err:[%v], msg:[%v]", err, message)
			continue
		}
		//更新资源doc内的cover字段
		ret, err := res_info.UpdateResCoverByResCenter(ctx, message.NewKey, message.SnapKey)
		if err != nil {
			xlog.ErrorC(ctx, "DealMessage.UpdateResCoverByResCenter failed, err:[%v], msg:[%v]", err, message)
			continue
		}
		if ret == 0 {
			xlog.ErrorC(ctx, "DealMessage.UpdateResCoverByResCenter failed, ret=0, resource may be not exist, msg:[%v]", message)
			continue
		}
		//删除旧资源
		path := common.GetSrcPath(message.OldKey)
		err = frame.DelBlackFrameVideo(ctx, path)
		if err != nil {
			xlog.ErrorC(ctx, "DealMessage.DelBlackFrameVideo failed, err:[%v], msg:[%v]", err, message)
			continue
		}
		destPath := common.GetDestPath(message.OldKey) + ".mp4"
		err = frame.DelBlackFrameVideo(ctx, destPath)
		if err != nil {
			xlog.ErrorC(ctx, "DealMessage.DelBlackFrameVideo failed, err:[%v], msg:[%s]", err, message)
			continue
		}
		//err = frame.DelBlackFrameSnap(ctx, message.SnapKey)
		//if err != nil {
		//	xlog.ErrorC(ctx, "DealMessage.DelBlackFrameVideo failed, err:[%v], msg:[%v]", err, message)
		//	continue
		//}
	}
	return consumer.ConsumeSuccess, nil
}

func StartQueue() {
	_ = SubScribe(api.TopicBlackFrame, "")
	//for i := 0; i < 4; i++ {
	//	go func() {
	//		_ = SubScribe(api.TopicBlackFrame, "")
	//	}()
	//}
}
