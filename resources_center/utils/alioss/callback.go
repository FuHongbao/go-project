package alioss

type OssCallBack struct {
	Bucket   string `json:"bucket" binding:"required"`
	Filename string `json:"filename" binding:"required"` //文件名即资源qid,后期更新为string类型
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`
	Height   int    `json:"height"`
	Width    int    `json:"width"`
	Format   string `json:"format"`
	//MyVar    string `json:"my_var" binding:"required"`
}
