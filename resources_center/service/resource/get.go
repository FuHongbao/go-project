package resource

import (
	"context"
	"fmt"
	"strconv"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByQid"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/dao/DaoByTag"
	resourceDao "xgit.xiaoniangao.cn/xngo/service/resources_center/dao/resource"
)

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

func ByIDs(ctx context.Context, IDs []string) (qDocs map[string]*api.XngResourceInfoDoc, err error) {
	c := len(IDs)
	ch := make(chan *api.XngResourceInfoDoc, c)
	for _, id := range IDs {
		go func(id string, ch chan *api.XngResourceInfoDoc) {
			qDoc, err1 := ByID(ctx, id)
			if err1 == nil {
				if qDoc == nil {
					idInt, _ := strconv.ParseInt(id, 10, 64)
					qDoc = &api.XngResourceInfoDoc{ResId: idInt, Cover: 0}
				}
				ch <- qDoc
				xlog.DebugC(ctx, "get doc into ch")
				return
			}
			xlog.ErrorC(ctx, "fail to get qdoc, id:%s, err:%v", id, err1)
		}(id, ch)
	}

	qDocs = map[string]*api.XngResourceInfoDoc{}
	for i := 0; i < c; i++ {
		select {
		case qDoc := <-ch:
			qDocs[strconv.FormatInt(qDoc.ResId, 10)] = qDoc
		case <-time.After(time.Millisecond * 200):
			xlog.ErrorC(ctx, "timeout get id, ids:%v", IDs)
			err = fmt.Errorf("timeout get id")
			return
		}
	}
	return
}

func ByQeTag(ctx context.Context, qeTag string) (qid string, err error) {
	qid, err = resourceDao.GetQeTagCache(qeTag)
	if err != nil {
		xlog.ErrorC(ctx, "Get Uploaded LocalCache from redis failed, err:%v, qetag:%v", err, qeTag)
		return
	}
	if qid == "" {
		//缓存不存在，从mongo查询
		doc, nerr := DaoByTag.GetDocByTag(qeTag)
		if nerr != nil {
			err = nerr
			return
		}
		if doc == nil { //mongo未查到直接返回
			return
		}
		//mongo查到则设置缓存
		qid = strconv.FormatInt(doc.Qid, 10)
		nerr = resourceDao.SetQeTagCache(qeTag, qid, api.CacheMiddleTime)
		if nerr != nil {
			return
		}
	}
	return
}

type GetIDsBatch struct {
	Ret int                     `json:"ret"`
	Doc *api.XngResourceInfoDoc `json:"doc"`
	Id  string                  `json:"id"`
}

func ByIDBatch(ctx context.Context, IDs []string) (qDocs map[string]*api.XngResourceInfoDoc, idsNotExist []string, err error) {
	qDocs = map[string]*api.XngResourceInfoDoc{}
	qDocs, idsNotFound, err := resourceDao.GetCacheBatch(ctx, IDs)
	if err != nil {
		xlog.ErrorC(ctx, "ByIDBatch.GetCacheBatch get resource doc err:%v", err)
		err = nil
	}
	xlog.DebugC(ctx, "ByIDBatch.find from cache, not found:[%v]", idsNotFound)
	c := len(idsNotFound)
	if c > 0 { //从DB查询
		ch := make(chan GetIDsBatch, c)
		for _, id := range idsNotFound {
			go func(id string, ch chan GetIDsBatch) {
				qid, err1 := strconv.ParseInt(id, 10, 64)
				if err1 != nil {
					xlog.ErrorC(ctx, "fail to get id, ID:%s, err:%v", qid, err)
					err = err1
					return
				}
				exist := 1
				qDoc, err1 := DaoByQid.GetDocByQid(qid)
				if err1 == nil {
					if qDoc == nil {
						exist = 0
						//idInt, _ := strconv.ParseInt(id, 10, 64)
						//qDoc = &api.XngResourceInfoDoc{ResId: idInt, Cover: 0}
					}
					data := GetIDsBatch{
						Ret: exist,
						Doc: qDoc,
						Id:  id,
					}
					ch <- data
					return
				}
				xlog.ErrorC(ctx, "fail to get qdoc, id:%s, err:%v", id, err1)
			}(id, ch)
		}
		for i := 0; i < c; i++ {
			select {
			case data := <-ch:
				xlog.DebugC(ctx, "ByIDBatch.find from db, id:[%v]", data.Id)
				if data.Ret == 0 {
					idsNotExist = append(idsNotExist, data.Id)
				} else {
					qDocs[data.Id] = data.Doc
				}
			case <-time.After(time.Millisecond * 400):
				xlog.ErrorC(ctx, "timeout get id, ids:%v", IDs)
				err = fmt.Errorf("timeout get id")
				return
			}
		}
	}
	go func() {
		if qDocs != nil {
			for id, doc := range qDocs {
				_ = resourceDao.SetCache(ctx, id, doc, api.CacheMiddleTime)
			}
		}
	}()
	return
}
