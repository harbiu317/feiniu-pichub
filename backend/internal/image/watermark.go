package image

import (
	"image"
	"image/color"
	"image/draw"
	"log"
	"os"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// 系统常见 CJK 字体位置。按顺序尝试；第一个能打开的使用。
var systemFontCandidates = []string{
	// Linux / fnOS (Debian)
	"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/opentype/noto/NotoSansCJKsc-Regular.otf",
	"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/truetype/wqy/wqy-microhei.ttc",
	"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc",
	"/usr/share/fonts/truetype/arphic/ukai.ttc",
	"/usr/share/fonts/truetype/arphic/uming.ttc",
	"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
	// macOS
	"/System/Library/Fonts/PingFang.ttc",
	"/System/Library/Fonts/STHeiti Medium.ttc",
	// Windows
	"C:/Windows/Fonts/msyh.ttc",
	"C:/Windows/Fonts/msyh.ttf",
	"C:/Windows/Fonts/simhei.ttf",
	"C:/Windows/Fonts/simsun.ttc",
}

// SetWatermarkFontPath 由 server 侧传入配置里的自定义字体路径。
var userFontPath string
var faceCache sync.Map // key: "path|size" -> font.Face

func SetWatermarkFontPath(p string) {
	userFontPath = p
}

func loadFont(size float64) font.Face {
	candidates := make([]string, 0, len(systemFontCandidates)+1)
	if userFontPath != "" {
		candidates = append(candidates, userFontPath)
	}
	candidates = append(candidates, systemFontCandidates...)

	for _, p := range candidates {
		if p == "" {
			continue
		}
		cacheKey := p + "|" + fmtFloat(size)
		if v, ok := faceCache.Load(cacheKey); ok {
			return v.(font.Face)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		face, err := buildFace(data, size)
		if err != nil {
			log.Printf("[watermark] 字体解析失败 %s: %v", p, err)
			continue
		}
		faceCache.Store(cacheKey, face)
		log.Printf("[watermark] 使用字体: %s (size=%g)", p, size)
		return face
	}
	return nil
}

func buildFace(data []byte, size float64) (font.Face, error) {
	// 先尝试 TTC（字体集合，Noto/微软雅黑 常见格式）
	if c, err := sfnt.ParseCollection(data); err == nil && c.NumFonts() > 0 {
		f, err := c.Font(0)
		if err == nil {
			return opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
		}
	}
	// 回退：单个 TTF/OTF
	f, err := sfnt.Parse(data)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
}

func fmtFloat(f float64) string {
	s := make([]byte, 0, 16)
	s = append(s, byte(int(f)))
	return string(s)
}

// drawWatermark 右下角绘制半透明文字水印。支持 CJK（需系统或用户配置的 TTF/TTC 字体）。
func drawWatermark(src image.Image, text string, opacity int) image.Image {
	if text == "" {
		return src
	}
	if opacity <= 0 {
		opacity = 60
	}
	if opacity > 100 {
		opacity = 100
	}
	alpha := uint8(255 * opacity / 100)

	b := src.Bounds()
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, src, b.Min, draw.Src)

	// 字号按图片尺寸自适应（长边的 3%，下限 14，上限 72）
	longSide := b.Dx()
	if b.Dy() > longSide {
		longSide = b.Dy()
	}
	size := float64(longSide) * 0.03
	if size < 14 {
		size = 14
	}
	if size > 72 {
		size = 72
	}

	face := loadFont(size)
	var advance int
	var asc int

	if face != nil {
		metrics := face.Metrics()
		asc = metrics.Ascent.Round()
		// 量文字宽度
		for _, r := range text {
			adv, ok := face.GlyphAdvance(r)
			if !ok {
				adv, _ = face.GlyphAdvance(' ')
			}
			advance += adv.Round()
		}
	} else {
		// 无字体可用：回退 basicfont（仅 ASCII）
		face = basicfont.Face7x13
		asc = 10
		advance = len(text) * 7
	}

	padding := int(size * 0.5)
	if padding < 10 {
		padding = 10
	}
	x := b.Max.X - advance - padding
	y := b.Max.Y - padding
	if x < b.Min.X {
		x = b.Min.X + 4
	}

	// 先画阴影增加可读性
	shadowCol := color.RGBA{0, 0, 0, alpha}
	drawString(rgba, face, text, x+2, y+2, shadowCol)

	fg := color.RGBA{255, 255, 255, alpha}
	drawString(rgba, face, text, x, y, fg)

	_ = asc
	return rgba
}

func drawString(dst *image.RGBA, face font.Face, text string, x, y int, col color.Color) {
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)},
	}
	d.DrawString(text)
}
