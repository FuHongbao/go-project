package url

import (
	"github.com/gin-gonic/gin"
	url2 "net/url"
	"time"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
	urlService "xgit.xiaoniangao.cn/xngo/service/resources_center/service/url"
)

// AlbumReq ..
type AlbumReq []struct {
	ID       string `json:"id" binding:"required"`
	IsStream int    `json:"is_stream"`
}

// AlbumResp ...
type AlbumResp struct {
	URLs map[string]RespURLWithM3u8 `json:"urls"`
}

// AlbumURL ...
func AlbumURL(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req AlbumReq
	if !xc.GetReqObject(&req) {
		return
	}
	total := time.Now()
	resp := &AlbumResp{URLs: make(map[string]RespURLWithM3u8)}

	for _, v := range req {
		if v.ID == "" {
			continue
		}
		var urlM3u8, urlM3u8Internal string
		key := v.ID
		url, urlInternal := urlService.GetAlbumURL(xc, key, 0)
		switch v.IsStream {
		case 1:
			key += "/index_0.m3u8"
		case 2:
			key += "/index.m3u8"
		}
		if v.IsStream != 0 {
			urlM3u8, urlM3u8Internal = urlService.GetAlbumURL(xc, key, v.IsStream)
		}
		url, urlM3u8 = tmpReplaceHost(v.ID, url, urlM3u8)
		resp.URLs[v.ID] = RespURLWithM3u8{URL: url, URLInternal: urlInternal, URLM3u8: urlM3u8, URLM3u8Internal: urlM3u8Internal}
	}
	xlog.DebugC(xc, "AlbumURL use time:[%v] total", time.Since(total))
	xc.ReplyOK(resp)
}

func tmpReplaceHost(key, url, urlM3u8 string) (retUrl string, retUrlM3u8 string) {
	vids := []string{
		"60055026000001317de6f28f",
		"5f82c47d00000162bb32f1ff",
		"600ced9700000131367a245d",
		"60174ab50000013886cf4b9c",
		"601805eb0000013886d03fef",
		"5faf7a930000010da1c88e6c",
		"6014cb920000013886cc90f0",
		"5ffef08a000001183ed9e38a",
		"2623417037",
		"600c0ac40000013136795b85",
		"6013bf1c0000013886cba437",
		"601b21c90000013136896a1c",
		"601bd6a20000013886d4d35e",
		"6008fcbe0000013886c0976b",
		"601ef81100000131368dd8b3",
		"601f18850000013886d82fe1",
		"833196295",
		"5ff3ec3200000138c2c4de13",
		"6010cbfd00000131367e56c1",
		"3696073296",
		"601dcc5000000131368c9304",
		"3070817604",
		"2613945575",
		"60054dfe00000154fa00ab9d",
		"6016d39a0000013886cf0ba8",
		"601dfeb80000013886d70f5c",
		"601250d400000131367fe858",
		"1781242281",
		"601be0d300000131368a9c16",
		"6010af8a00000131367e38e6",
		"601a56f40000013886d29fd8",
		"5fb8b8a00000010da1d6d91a",
		"5ffc1fbb0000014328464b94",
		"600e9d330000013886c67ee8",
		"6002b08900000123218689dc",
		"60077c0c0000013886bee959",
		"6021f0810000013136913c77",
		"5ff9000e00000164f9732300",
		"5ff57b870000017909040361",
		"5ee8d64d000001083955c65e",
		"5fd47f7b000001701b9778f1",
		"5feddc7800000144ce78aee1",
		"601cfa5600000131368bc5ae",
		"5ff3cd30000001016050f648",
		"601ece410000013886d813d8",
		"601bc19200000131368a6b69",
		"1749976378",
		"5ed44cee0000017fd23ec237",
		"601a7d7a00000131368887dd",
		"5fd4d9350000014f1232d752",
		"2223048995",
		"5fea95f30000015181969cd1",
		"5fd001a9000001701b91a510",
		"6014c3ab000001313682386b",
		"601267aa0000013886ca579e",
		"6013c06100000131368154a7",
		"600c34260000013886c3f87f",
		"5ffeecd7000001183ed9dd8c",
		"6005a95100000154fa012ed9",
		"3684181759",
		"600fb24b00000131367d3c75",
		"5fea90e100000170ab0d84da",
		"2830864853",
		"6004bc5f000001232188a2b3",
		"2258130454",
		"2486886622",
		"600a8653000001313677cf45",
		"6019fba40000013886d221ed",
		"5fdf3d4500000130ed177243",
		"601f32960000013886d84b20",
		"60169319000001313684538d",
		"600ced960000013886c47c89",
		"5fe9bed000000170ab0c9a34",
		"5fd2b6e5000001701b95335d",
		"600fc5840000013886c7abf8",
		"5fe6c77800000170ab08accd",
		"60083e920000013886c01189",
		"6018dcd00000013136869c95",
		"600fce2f00000131367d6260",
		"601689620000013886ce92cc",
		"5fff28fc0000017fc8a3918d",
		"601815ae0000013136860b70",
		"5fe9889e00000170ab0c40f7",
		"2205936519",
		"601141c90000013886c93ec5",
		"6007892d0000013886bf0235",
		"3692326879",
		"6011ff960000013886c9e008",
		"600c32ee0000013886c3f6bf",
		"5eb2aac000000150f28453f8",
		"2959728608",
		"600397310000017fc8a81960",
		"6012b4700000013136806e3f",
		"5fd35b6c0000014f1230d5ce",
		"5ff1816a000001214c1dbe3c",
		"600d31af00000131367a7b4a",
		"60068043000001317de813f5",
		"6014a96d0000013136821f80",
		"5fbce51900000162bb910200",
		"2987038062",
	}
	retUrl = url
	retUrlM3u8 = urlM3u8
	for _, id := range vids {
		if key == id && conf.Env == lib.PROD {
			u, err := url2.Parse(url)
			if err != nil {
				return
			}
			u.Scheme = "https"
			u.Host = "cdn-xalbum-mp4.xiaoniangao.cn"
			retUrl = u.String()

			if urlM3u8 != "" {
				u2, err := url2.Parse(urlM3u8)
				if err != nil {
					return
				}
				u2.Scheme = "https"
				u2.Host = "cdn-xalbum-m3u8.xiaoniangao.cn"
				retUrlM3u8 = u2.String()
			}
			break
		}
	}
	return
}
