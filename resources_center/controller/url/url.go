package url

type RespURL struct {
	URL         string `json:"url"`
	URLInternal string `json:"url_internal"`
}
type RespURLWithM3u8 struct {
	URL             string `json:"url"`
	URLInternal     string `json:"url_internal"`
	URLM3u8         string `json:"url_m3u8,omitempty"`
	URLM3u8Internal string `json:"url_m3u8_internal,omitempty"`
}
type RespURLs struct {
	URLs               []string `json:"urls"`
	VideoUrl           string   `json:"video_url,omitempty"`
	URLsNoMark         []string `json:"urls_no_mark,omitempty"`
	URLInternals       []string `json:"url_internals"`
	VideoUrlInternal   string   `json:"video_url_internal,omitempty"`
	URLsNoMarkInternal []string `json:"urls_no_mark_internal,omitempty"`
}
type RespURLWithM3u8HitExp struct {
	IsHit           bool   `json:"is_hit"`
	URL             string `json:"url"`
	URLInternal     string `json:"url_internal"`
	URLM3u8         string `json:"url_m3u8,omitempty"`
	URLM3u8Internal string `json:"url_m3u8_internal,omitempty"`
}
