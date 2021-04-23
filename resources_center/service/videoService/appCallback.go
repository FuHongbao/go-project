package videoService

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/mts"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByQid"
	resourceDao "xgit.xiaoniangao.cn/xngo/service/resources_center/dao/resource"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/utils/alioss"
)

const (
	VideoResCoverType = "img"
)

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
	Trans    *int    `json:"trans,omitempty"`
}

func GetCallbackInfo(qDoc *api.XngResourceInfoDoc, resId int64) (resp *MediaInfoResp) {
	resp = &MediaInfoResp{
		Id:    resId,
		Qid:   qDoc.ResId,
		Ty:    qDoc.Type,
		Size:  qDoc.Size,
		Upt:   qDoc.Upt,
		Mt:    qDoc.Mt,
		Ct:    qDoc.Ct,
		Src:   qDoc.Src,
		Fmt:   qDoc.Fmt,
		W:     qDoc.W,
		H:     qDoc.H,
		Du:    qDoc.Du,
		Cover: qDoc.Cover,
		Code:  qDoc.Code,
		QeTag: qDoc.QeTag,
		Trans: qDoc.TransCode,
	}
	return
}

func GetVideoResBaseDoc(ctx context.Context, stsData *StsInfo, key, qeTag string) (doc *api.XngResourceInfoDoc, err error) {
	endPoint := stsData.Data.Endpoint
	regionID := endPoint[4 : len(endPoint)-13]
	client, err := mts.NewClientWithStsToken(regionID, stsData.Data.AccessKey, stsData.Data.SecretKey, stsData.Data.SecurityToken)
	if err != nil {
		xlog.ErrorC(ctx, "create new client with stsToken error:%v", err)
		return
	}
	if conf.Env == lib.PROD {
		client.Domain = api.MtsVpcDomain
	}
	uploadBucket := stsData.Data.InputBucket
	loc := endPoint[:len(endPoint)-13]
	doc, err = GetVideoInfo(client, uploadBucket, loc, key)
	if err != nil {
		xlog.ErrorC(ctx, "get media information error, key:%v, error:%v", key, err)
		return
	}
	if doc == nil {
		xlog.ErrorC(ctx, "get media information error, key:%v, doc is nil", key)
		err = errors.New("media doc is nil")
		return
	}
	doc.QeTag = qeTag
	return
}

type ImgInfoResp struct {
	FileSize  map[string]string `json:"FileSize"`
	Format    map[string]string `json:"Format"`
	ImgHeight map[string]string `json:"ImageHeight"`
	ImgWidth  map[string]string `json:"ImageWidth"`
}

func GetImgInfo(ctx context.Context, filename string, resType int) (doc *api.XngResourceInfoDoc, err error) {

	url := alioss.GetImageSignURL(ctx, filename, []alioss.ImageAction{&alioss.Info{}})
	client := &http.Client{}
	imgResp, err := client.Get(url)
	if err != nil {
		return
	}
	defer func() {
		if imgResp != nil && imgResp.Body != nil {
			_ = imgResp.Body.Close()
		}
	}()
	body, err := ioutil.ReadAll(imgResp.Body)
	resp := ImgInfoResp{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		xlog.ErrorC(ctx, "fail to unmarshal imageInfo, url:%s, resp:%v, err:%v", url, fmt.Sprintf("%v", imgResp.Body), err)
		return
	}

	resId, err := strconv.ParseInt(filename, 10, 64)
	if err != nil {
		return
	}
	size, err := strconv.ParseInt(resp.FileSize["value"], 10, 64)
	if err != nil {
		return
	}
	width, err := strconv.Atoi(resp.ImgWidth["value"])
	if err != nil {
		return
	}
	height, err := strconv.Atoi(resp.ImgHeight["value"])
	if err != nil {
		return
	}
	upt := time.Now().UnixNano() / 1e6
	doc = &api.XngResourceInfoDoc{
		ResId: resId,
		Type:  resType,
		Size:  size,
		Upt:   upt,
		Mt:    upt,
		Fmt:   resp.Format["value"],
		Ort:   1,
		W:     width,
		H:     height,
		Ref:   1,
	}
	xlog.InfoC(ctx, "get snap image info, resp:%v, doc:%v", resp, doc)
	return
}

func ByID(ctx context.Context, ID string) (qDoc *api.XngResourceInfoDoc, err error) {
	ret, err := resourceDao.GetCache(ctx, ID)
	if err != nil {
		xlog.ErrorC(ctx, "get resource err:%v, id:%s", err, ID)
		err = nil
	}

	if ret != nil {
		qDoc = ret
	} else {
		id, err1 := strconv.ParseInt(ID, 10, 64)
		if err1 != nil {
			xlog.ErrorC(ctx, "fail to get id, ID:%s, err:%v", ID, err)
			err = err1
			return
		}

		qDoc, err1 = DaoByQid.GetDocByQid(id)
		if err1 != nil {
			return nil, err1
		}
	}

	if qDoc != nil {
		_ = resourceDao.SetCache(ctx, ID, qDoc, api.CacheMiddleTime)
	}

	return
}
