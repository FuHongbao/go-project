package alioss

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ImageAction 阿里云图片操作参数 https://help.aliyun.com/document_detail/44687.html?spm=a2c4g.11186623.6.1253.72fac1f69rxxBQ
type ImageAction interface {
	ToString() string
}

// Resize 图片缩放操作
type Resize struct {
	Mode       string // 缩放的模式，lfit、mfit、fill、pad、fixed
	Width      int
	Height     int
	P          int   // 倍数百分比。 小于 100，即是缩小，大于 100 即是放大。1-1000
	WatermarkP int   // 水印时有用，倍数百分比。 小于 100，即是缩小，大于 100 即是放大。1-1000
	Limit      *int8 // 指定当目标缩略图大于原图时是否处理。值是 1 表示不处理；值是 0 表示处理。
	// 文档中还有其他效果，可以根据文档继续扩充
}

// ToString resize的格式化参数
func (r *Resize) ToString() string {
	s := ""
	if r.Mode != "" {
		s = s + ",m_" + r.Mode
	}
	if r.Width != 0 {
		s = fmt.Sprintf("%s,w_%d", s, r.Width)
	}
	if r.Height != 0 {
		s = fmt.Sprintf("%s,h_%d", s, r.Height)
	}
	if r.P != 0 {
		s = fmt.Sprintf("%s,p_%d", s, r.P)
	}
	if r.WatermarkP != 0 {
		s = fmt.Sprintf("%s,P_%d", s, r.WatermarkP)
	}
	if r.Limit != nil && *r.Limit == 0 {
		s = fmt.Sprintf("%s,limit_%d", s, *r.Limit)
	}
	if s != "" {
		s = fmt.Sprintf("/resize%s", s)
	}
	return s
}

// Crop 裁剪操作
type Crop struct {
	Width   int
	Height  int
	gravity string
	LocateX int
	LocateY int
	// 文档中还有其他效果，可以根据文档继续扩充
}

// ToString 格式化参数
func (c *Crop) ToString() string {
	s := ""
	if c.LocateX != 0 {
		s = fmt.Sprintf("%s,x_%d", s, c.LocateX)
	}
	if c.LocateY != 0 {
		s = fmt.Sprintf("%s,y_%d", s, c.LocateY)
	}
	if c.Width != 0 {
		s = fmt.Sprintf("%s,w_%d", s, c.Width)
	}
	if c.Height != 0 {
		s = fmt.Sprintf("%s,h_%d", s, c.Height)
	}
	if c.gravity != "" {
		s = fmt.Sprintf("%s,g_%s", s, c.gravity)
	}
	if s != "" {
		s = fmt.Sprintf("/crop%s", s)
	}
	return s
}

// Rotate 旋转操作
type Rotate struct {
	Value int // 图片按顺时针旋转的角度,[0, 360]默认值为 0，表示不旋转。
}

// ToString 格式化参数
func (r *Rotate) ToString() string {
	s := ""
	if r.Value != 0 {
		s = fmt.Sprintf("%s,%d", s, r.Value)
	}
	if s != "" {
		s = fmt.Sprintf("/rotate%s", s)
	}
	return s
}

// Blur 模糊操作
type Blur struct {
	Radius int // 模糊半径，[1,50]r 越大图片越模糊。
	S      int // 正态分布的标准差，[1,50]s 越大图片越模糊。
}

// ToString 格式化参数
func (b *Blur) ToString() string {
	s := ""
	if b.Radius != 0 {
		s = fmt.Sprintf("%s,r_%d", s, b.Radius)
	}
	if b.S != 0 {
		s = fmt.Sprintf("%s,s_%d", s, b.S)
	}
	if s != "" {
		s = fmt.Sprintf("/blur%s", s)
	}
	return s
}

// Format 格式化
type Format struct {
	Value string
}

// ToString 格式化参数
func (f *Format) ToString() string {
	s := ""
	if f.Value != "" {
		s = fmt.Sprintf("%s,%s", s, f.Value)
	}
	if s != "" {
		s = fmt.Sprintf("/format%s", s)
	}
	return s
}

// Interlace 渐进显示 此参数只有当效果图是 jpg 格式时才有意义
type Interlace struct {
	Value int8
}

// ToString 格式化参数
func (i *Interlace) ToString() string {
	s := ""
	if i.Value != 0 {
		s = fmt.Sprintf("%s,%d", s, i.Value)
	}
	if s != "" {
		s = fmt.Sprintf("/interlace%s", s)
	}
	return s
}

// Quality 质量
type Quality struct {
	Relative int8 // 相对质量[1,100]
	Absolute int8 // 绝对质量[1,100]
}

// ToString 格式化参数
func (q *Quality) ToString() string {
	s := ""
	if q.Relative != 0 {
		s = fmt.Sprintf("%s,q_%d", s, q.Relative)
	}
	if q.Absolute != 0 {
		s = fmt.Sprintf("%s,Q_%d", s, q.Absolute)
	}
	if s != "" {
		s = fmt.Sprintf("/quality%s", s)
	}
	return s
}

// Circle 内切圆裁剪图片 https://help.aliyun.com/document_detail/44695.html?spm=a2c4g.11186623.6.1661.4b885bbffmovir
type Circle struct {
	R int // 指定裁剪图片所用的圆形区域的半径
}

// ToString 格式化参数
func (c *Circle) ToString() string {
	s := ""
	if c.R != 0 {
		s = fmt.Sprintf("%s,r_%d", s, c.R)
	}
	if s != "" {
		s = fmt.Sprintf("/circle%s", s)
	}
	return s
}

// Info 获取信息 https://help.aliyun.com/document_detail/44975.html?spm=a2c4g.11186623.6.1437.b122218cXc7Zi8
type Info struct {
}

// ToString 格式化参数
func (i *Info) ToString() string {
	return "/info"
}

// Watermark 水印 https://help.aliyun.com/document_detail/44957.html?spm=a2c4g.11186623.6.1273.1073e8493seKe0
type Watermark struct {
	Image        string // 必选参数
	Transparency int    // 透明度
	Gravity      string // 位置，水印打在图的位置，详情参考下方区域数值对应图。 取值范围：[nw,north,ne,west,center,east,sw,south,se]
	X            *int
	Y            *int
	VOffset      int
	Text         string // 文字
	Ty           string // 字体
	Color        string //字体颜色
	Size         int    //大小(0,1000]，默认值：40
	// 文档中还有其他效果，可以根据文档继续扩充
}

// ToString 格式化参数
func (w *Watermark) ToString() string {
	s := ""
	if w.Image == "" && w.Text == "" {
		return s
	}

	if w.Image != "" {
		s = fmt.Sprintf("%s,image_%s", s, w.Image)
	}
	if w.Text != "" {
		s = fmt.Sprintf("%s,text_%s", s, w.Text)
	}

	if w.Transparency != 0 {
		s = fmt.Sprintf("%s,t_%d", s, w.Transparency)
	}
	if w.Gravity != "" {
		s = fmt.Sprintf("%s,g_%s", s, w.Gravity)
	}
	if w.X != nil {
		s = fmt.Sprintf("%s,x_%d", s, *w.X)
	}
	if w.Y != nil {
		s = fmt.Sprintf("%s,y_%d", s, *w.Y)
	}
	if w.Ty != "" {
		s = fmt.Sprintf("%s,type_%s", s, w.Ty)
	}
	if w.Color != "" {
		s = fmt.Sprintf("%s,color_%s", s, w.Color)
	}
	if w.Size != 0 {
		s = fmt.Sprintf("%s,size_%d", s, w.Size)
	}
	if s != "" {
		s = fmt.Sprintf("/watermark%s", s)
	}
	return s
}

// GetDegree 根据ort 获得旋转角度
func GetDegree(ort int8) int {
	switch ort {
	case 6:
		return 90
	case 8:
		return 270
	case 3:
		return 180
	default:
		return 0
	}
}

// ParseQs 根据qs解析出图片的参数
func ParseQs(qs string) (actions []ImageAction) {
	// imageMogr2/gravity/center/rotate/$/thumbnail/!165x165r/crop/165x165/interlace/1/format/jpg
	arr := strings.Split(qs, "/")
	//xlog.Debug("qs:%s, split by '/', arr:%v", qs, arr)
	reg := regexp.MustCompile(`\d+`)
	var gravity string
	actions = []ImageAction{}
	for i := 0; i+1 < len(arr); i++ {
		switch arr[i] {
		case "thumbnail":
			// 图片缩放
			r := getResize(arr[i+1])
			actions = append(actions, r)
		case "gravity":
			// 裁剪位置
			g := arr[i+1]
			gravity = getGravity(g)
		case "crop":
			// 图片裁剪
			finds := reg.FindAllString(arr[i+1], -1)
			if len(finds) == 2 {
				c := &Crop{}
				c.Width, _ = strconv.Atoi(finds[0])
				c.Height, _ = strconv.Atoi(finds[1])
				if gravity != "" {
					c.gravity = gravity
				}
				actions = append(actions, c)
			} else if len(finds) == 4 {
				c := &Crop{}
				c.Width, _ = strconv.Atoi(finds[0])
				c.Height, _ = strconv.Atoi(finds[1])
				c.LocateX, _ = strconv.Atoi(finds[2])
				c.LocateY, _ = strconv.Atoi(finds[3])
				if gravity != "" {
					c.gravity = gravity
				}
				actions = append(actions, c)
			}
		case "quality":
			// 图片质量 https://help.aliyun.com/document_detail/44705.html?spm=a2c4g.11186623.6.1272.d53a57f9HCH7HJ
			q := &Quality{}
			r, _ := strconv.Atoi(arr[i+1])
			q.Relative = int8(r)
			actions = append(actions, q)
		case "format":
			// 图片格式 https://help.aliyun.com/document_detail/44703.html?spm=a2c4g.11186623.6.1270.5379e849AfcmI2
			f := &Format{Value: arr[i+1]}
			actions = append(actions, f)
		case "interlace":
			// 图片显示效果 https://help.aliyun.com/document_detail/44704.html?spm=a2c4g.11186623.6.1271.55302f66dHS9hv
			v, _ := strconv.Atoi(arr[i+1])
			i := &Interlace{Value: int8(v)}
			actions = append(actions, i)
		case "rotate":
			//旋转 https://help.aliyun.com/document_detail/44690.html?spm=a2c4g.11186623.6.1416.d53a57f9RwBEZm
			if arr[i+1] != "$" {
				v, _ := strconv.Atoi(arr[i+1])
				actions = append(actions, &Rotate{Value: v})
			}
		case "circle":
			c := &Circle{}
			c.R, _ = strconv.Atoi(arr[i+1])
			actions = append(actions, c)
		}
	}
	auto := 0
	if strings.Contains(qs, "auto-orient") {
		auto = 1
	} else {
		auto = 0
	}
	ao := &AutoOrient{Value: auto}
	actions = append(actions, ao)

	//if strings.Contains(qs, "auto-orient") {
	//	auto = 1
	//	ao := &AutoOrient{Value: auto}
	//	actions = append(actions, ao)
	//} else if strings.Contains(qs, "no-orient") {
	//	ao := &AutoOrient{Value: auto}
	//	actions = append(actions, ao)
	//}
	return
}

// 获取Gravity信息
func getGravity(g string) string {
	gravity := ""
	g = strings.ToLower(g)
	switch g {
	case "northwest":
		gravity = "nw"
	case "north":
		gravity = "north"
	case "northeast":
		gravity = "ne"
	case "west":
		gravity = "west"
	case "center":
		gravity = "center"
	case "east":
		gravity = "east"
	case "southwest":
		gravity = "sw"
	case "south":
		gravity = "south"
	case "southeast":
		gravity = "se"
	}
	return gravity
}

// 获取resize信息
func getResize(s string) (r *Resize) {
	r = &Resize{}
	reg := regexp.MustCompile(`([!]?)(\d*)([x]?)(\d*)([!pxr]{0,2})`)
	finds := reg.FindStringSubmatch(s)
	if len(finds) != 6 {
		return
	}
	// !50p 按照倍数缩放
	if finds[1] == "!" && finds[5] == "p" {
		r.P, _ = strconv.Atoi(finds[2])
		return
	}

	var i int8
	// 100x | x200 |  100x200
	if finds[3] == "x" && finds[1] == "" && finds[5] == "" {
		if finds[2] != "" {
			r.Width, _ = strconv.Atoi(finds[2])
		}
		if finds[4] != "" {
			r.Height, _ = strconv.Atoi(finds[4])
		}
		r.Limit = &i
		return
	}

	// !100x200r
	if finds[1] == "!" && finds[5] == "r" {
		r.Mode = "mfit"
		r.Width, _ = strconv.Atoi(finds[2])
		r.Height, _ = strconv.Atoi(finds[4])
		r.Limit = &i
		return
	}
	return
}

// GetWatermarkData 获取水印data
func GetWatermarkData(ctx context.Context, key string, actions []ImageAction) (data string) {
	s := fmt.Sprintf("%s?x-oss-process=image", key)
	for _, a := range actions {
		s = s + a.ToString()
	}
	data = base64.StdEncoding.EncodeToString([]byte(s))
	data = strings.ReplaceAll(data, "+", "-")
	data = strings.ReplaceAll(data, "/", "_")
	return
}

// ParseWatermarkQs 根据qs解析出图片的参数
func ParseWatermarkQs(qs string) *Watermark {
	// watermark/dissolve/<dissolve>/gravity/<gravity>/dx/<distanceX>/dy/<distanceY>/ws/<watermarkScale>/wst/<watermarkScaleType>
	arr := strings.Split(qs, "/")
	w := &Watermark{}
	for i := 0; i+1 < len(arr); i++ {
		switch arr[i] {
		case "dissolve":
			// 透明度
			t := arr[i+1]
			w.Transparency, _ = strconv.Atoi(t)
		case "gravity":
			// 位置
			g := arr[i+1]
			w.Gravity = getGravity(g)
		case "dx":
			// 横轴边距
			x := arr[i+1]
			v, _ := strconv.Atoi(x)
			w.X = &v
		case "dy":
			// 纵轴边距
			y := arr[i+1]
			v, _ := strconv.Atoi(y)
			w.Y = &v
		case "font":
			w.Ty = arr[i+1]
		case "fontsize":
			s := arr[i+1]
			w.Size, _ = strconv.Atoi(s)
		case "fill":
			w.Color = arr[i+1]
		}
	}

	return w
}

// AutoOrient 自适应方向 https://help.aliyun.com/document_detail/44691.html?spm=a2c4g.11186623.6.1415.5379e849Z52pH2
type AutoOrient struct {
	Value int // 进行自动旋转  0：表示按原图默认方向，不进行自动旋转;  1（默认）：先进行图片旋转，然后再进行缩略。
}

// ToString 格式化参数
func (o *AutoOrient) ToString() string {
	return fmt.Sprintf("/auto-orient,%d", o.Value)
}
