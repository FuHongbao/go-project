package url

import (
	"context"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/conf"
)

// cdn的类型
const (
	TypeQiNiu      = 1
	TypeAliOSS     = 2   // 阿里
	TypeTengXun    = 3   // 腾讯
	TypeAliForeign = 102 // 阿里海外
	TypeAliOSSPK   = 202 // 阿里oss另一个账户流量包
)
const (
	TypeInternalResourceAli = "resource_ali"
	TypeInternalAlbumAli    = "album_ali"
	TypeInternalLiveAli     = "live_ali"
	TypeInternalAuditMusic  = "audit_music_ali"
	TypeInternalStaticAli   = "static_ali"
	TypeInternalGuideAli    = "live_guide_ali"
	TypeInternalAppLog      = "app_log_ali"
)

func getCDNDomain(ctx context.Context, cdnType string) string {
	domain, ok := conf.C.Cdn[cdnType]
	if !ok {
		xlog.ErrorC(ctx, "fail to get img domain, cdn:%s", cdnType)
		return ""
	}
	return domain
}
func getInternalDomain(ctx context.Context, cdnType string) string {
	domain, ok := conf.C.Internal[cdnType]
	if !ok {
		xlog.ErrorC(ctx, "fail to get internal domain, internal:%s", cdnType)
		return ""
	}
	return domain
}

// 获取cdn domain
func getAlbumCDNDomain(ctx context.Context, cdnTy int64) string {
	s := ""
	switch cdnTy {
	case TypeQiNiu:
		s = ""
	case TypeAliOSS:
		s = "album_ali"
	case TypeTengXun:
		s = "album_tengxun"
	case TypeAliForeign:
		s = ""
	case TypeAliOSSPK:
		s = ""
	}
	domain := getCDNDomain(ctx, s)
	return domain
}

// 获取cdn domain
func getLiveCDNDomain(ctx context.Context, cdnTy int64) string {
	s := ""
	switch cdnTy {
	case TypeQiNiu:
		s = ""
	case TypeAliOSS:
		s = "live_ali"
	case TypeTengXun:
		s = "live_tengxun"
	case TypeAliForeign:
		s = ""
	case TypeAliOSSPK:
		s = ""
	}
	domain := getCDNDomain(ctx, s)
	return domain
}
