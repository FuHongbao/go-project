package api

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
type XngResourceInfoDoc struct {
	ResId     int64   `json:"_id" bson:"_id"`
	Type      int     `json:"ty" bson:"ty,omitempty"`
	Size      int64   `json:"size" bson:"size,omitempty"`
	QeTag     string  `json:"qetag" bson:"qetag,omitempty"`
	Upt       int64   `json:"upt" bson:"upt,omitempty"`
	Mt        int64   `json:"mt" bson:"mt,omitempty"`
	Ct        int64   `json:"ct,omitempty" bson:"ct,omitempty"`
	Src       string  `json:"src" bson:"src,omitempty"`
	Fmt       string  `json:"fmt" bson:"fmt,omitempty"`
	Ort       int     `json:"ort" bson:"ort,omitempty"`
	W         int     `json:"w" bson:"w,omitempty"`
	H         int     `json:"h" bson:"h,omitempty"`
	Ref       int     `json:"ref" bson:"ref,omitempty"`
	Du        float64 `json:"du,omitempty" bson:"du,omitempty"`
	Cover     int64   `json:"cover,omitempty" bson:"cover,omitempty"`
	CoverTp   string  `json:"covertp,omitempty" bson:"covertp,omitempty"`
	Code      string  `json:"code,omitempty" bson:"code,omitempty"`
	TransCode *int    `json:"trans,omitempty" bson:"trans"`
}

type BlackFrameMqMessage struct {
	OldKey    string  `json:"old_key"`
	NewKey    string  `json:"new_key"`
	SnapKey   string  `json:"snap_key"`
	Prod      int     `json:"prod"`
	Proj      string  `json:"proj"`
	FrameTime float64 `json:"frame_time"`
	Url       string  `json:"url"`
}

const (
	ResourceTypeVideo   = 6
	CommUploadConfigURL = "http://192.168.11.50:8987/resource/get_upload_info"
	ReplaceCoverURL     = "http://192.168.11.50:8987/snap/replace_snap"
)
const (
	TopicBlackFrame = "black_frames"
)
