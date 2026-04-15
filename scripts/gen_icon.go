//go:build ignore

// 生成 PicHub 应用图标。
// 设计：紫蓝渐变圆角方形 + 白色相片角标（山峰 + 太阳）+ 右上角"+"小圆。
// 输出：ICON.PNG / ICON_256.PNG / app/ui/images/icon_256.png / icon_64.png
//
// 运行：go run scripts/gen_icon.go
package main

import (
	"flag"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

type RGBA = color.RGBA

var (
	colBgTop    = RGBA{94, 106, 210, 255}  // #5E6AD2 Linear 紫
	colBgBot    = RGBA{139, 92, 246, 255}  // #8B5CF6
	colFg       = RGBA{255, 255, 255, 255}
	colFgShadow = RGBA{0, 0, 0, 38}
	colPlusBg   = RGBA{255, 255, 255, 255}
	colPlusFg   = RGBA{94, 106, 210, 255}
)

func main() {
	outDir := flag.String("out", ".", "输出目录")
	flag.Parse()

	size := 1024
	img := drawIcon(size)

	// 保存几种尺寸
	save(img, filepath.Join(*outDir, "icon_1024.png"))
	save(imaging.Resize(img, 512, 512, imaging.Lanczos), filepath.Join(*outDir, "icon_512.png"))
	save(imaging.Resize(img, 256, 256, imaging.Lanczos), filepath.Join(*outDir, "icon_256.png"))
	save(imaging.Resize(img, 128, 128, imaging.Lanczos), filepath.Join(*outDir, "icon_128.png"))
	save(imaging.Resize(img, 64, 64, imaging.Lanczos), filepath.Join(*outDir, "icon_64.png"))

	log.Println("✓ 图标已生成到", *outDir)
}

func save(im image.Image, path string) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, im); err != nil {
		log.Fatal(err)
	}
}

// drawIcon 绘制 size x size 的图标。
func drawIcon(size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))

	// 1. 圆角背景 + 竖向渐变
	radius := size * 22 / 100
	drawRoundedGradient(dst, radius, colBgTop, colBgBot)

	// 2. 中央相片元素：白色圆角矩形 + 山 + 太阳
	pw := size * 58 / 100
	ph := size * 48 / 100
	px := (size - pw) / 2
	py := (size-ph)/2 + size*3/100
	drawPicture(dst, px, py, pw, ph)

	// 3. 右上角"+"徽章
	br := size * 18 / 100
	bx := size - br - size*8/100
	by := size * 8 / 100
	drawPlusBadge(dst, bx+br/2, by+br/2, br/2)

	return dst
}

// drawRoundedGradient 画圆角矩形 + 竖向渐变底色。
func drawRoundedGradient(dst *image.RGBA, r int, top, bot RGBA) {
	b := dst.Bounds()
	w, h := b.Dx(), b.Dy()
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h-1)
		col := lerp(top, bot, t)
		for x := 0; x < w; x++ {
			if !insideRoundedRect(x, y, w, h, r) {
				continue
			}
			dst.SetRGBA(x, y, col)
		}
	}
}

func insideRoundedRect(x, y, w, h, r int) bool {
	if x >= r && x < w-r {
		return y >= 0 && y < h
	}
	if y >= r && y < h-r {
		return x >= 0 && x < w
	}
	// 四个角
	cx, cy := r, r
	if x >= w-r {
		cx = w - r - 1
	}
	if y >= h-r {
		cy = h - r - 1
	}
	dx := float64(x - cx)
	dy := float64(y - cy)
	return dx*dx+dy*dy <= float64(r*r)
}

func lerp(a, b RGBA, t float64) RGBA {
	return RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 255,
	}
}

// drawPicture 画一个"相片"：白色底板，山峰 + 太阳。
func drawPicture(dst *image.RGBA, x, y, w, h int) {
	// 白色卡片（带轻微阴影）
	shadowOffset := h / 28
	shadow := image.NewRGBA(image.Rect(0, 0, w, h))
	for yy := 0; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			if insideRoundedRect(xx, yy, w, h, h/9) {
				shadow.SetRGBA(xx, yy, colFgShadow)
			}
		}
	}
	draw.Draw(dst, image.Rect(x, y+shadowOffset, x+w, y+h+shadowOffset), shadow, image.Point{}, draw.Over)

	card := image.NewRGBA(image.Rect(0, 0, w, h))
	for yy := 0; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			if insideRoundedRect(xx, yy, w, h, h/9) {
				card.SetRGBA(xx, yy, colFg)
			}
		}
	}

	// 太阳：左上
	sunR := h / 7
	sunCx := w * 28 / 100
	sunCy := h * 32 / 100
	drawDisc(card, sunCx, sunCy, sunR, RGBA{255, 205, 93, 255})

	// 两座山：右侧三角
	mountain := RGBA{94, 106, 210, 255}
	mtn2 := RGBA{139, 92, 246, 255}
	// 后山（高）
	drawTriangle(card,
		w*52/100, h*30/100,
		w*24/100, h*82/100,
		w*80/100, h*82/100,
		mtn2,
	)
	// 前山（低）
	drawTriangle(card,
		w*70/100, h*45/100,
		w*48/100, h*82/100,
		w*92/100, h*82/100,
		mountain,
	)

	// 底部"地面"一条
	for yy := h * 82 / 100; yy < h*82/100+h/50; yy++ {
		for xx := w / 12; xx < w-w/12; xx++ {
			if insideRoundedRect(xx, yy, w, h, h/9) {
				card.SetRGBA(xx, yy, RGBA{94, 106, 210, 255})
			}
		}
	}

	draw.Draw(dst, image.Rect(x, y, x+w, y+h), card, image.Point{}, draw.Over)
}

func drawDisc(dst *image.RGBA, cx, cy, r int, c RGBA) {
	for yy := cy - r; yy <= cy+r; yy++ {
		for xx := cx - r; xx <= cx+r; xx++ {
			dx := xx - cx
			dy := yy - cy
			d := math.Sqrt(float64(dx*dx + dy*dy))
			if d <= float64(r) {
				// 边缘抗锯齿
				a := 1.0
				if d > float64(r)-1 {
					a = float64(r) - d
				}
				if a < 0 {
					a = 0
				}
				if xx >= 0 && yy >= 0 && xx < dst.Bounds().Dx() && yy < dst.Bounds().Dy() {
					orig := dst.RGBAAt(xx, yy)
					blend := lerpRGBA(orig, c, a)
					dst.SetRGBA(xx, yy, blend)
				}
			}
		}
	}
}

func lerpRGBA(a, b RGBA, t float64) RGBA {
	if t >= 1 {
		return b
	}
	return RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 255,
	}
}

// drawTriangle 使用扫描线填充三角形（三点任意顺序）。
func drawTriangle(dst *image.RGBA, x1, y1, x2, y2, x3, y3 int, c RGBA) {
	minX := min3(x1, x2, x3)
	maxX := max3(x1, x2, x3)
	minY := min3(y1, y2, y3)
	maxY := max3(y1, y2, y3)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if pointInTriangle(x, y, x1, y1, x2, y2, x3, y3) {
				if x >= 0 && y >= 0 && x < dst.Bounds().Dx() && y < dst.Bounds().Dy() {
					dst.SetRGBA(x, y, c)
				}
			}
		}
	}
}

func pointInTriangle(px, py, x1, y1, x2, y2, x3, y3 int) bool {
	d1 := sign(px, py, x1, y1, x2, y2)
	d2 := sign(px, py, x2, y2, x3, y3)
	d3 := sign(px, py, x3, y3, x1, y1)
	hasNeg := d1 < 0 || d2 < 0 || d3 < 0
	hasPos := d1 > 0 || d2 > 0 || d3 > 0
	return !(hasNeg && hasPos)
}

func sign(px, py, x1, y1, x2, y2 int) int {
	return (px-x2)*(y1-y2) - (x1-x2)*(py-y2)
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
func max3(a, b, c int) int {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}

// drawPlusBadge 画一个白色圆 + 紫色"+"。
func drawPlusBadge(dst *image.RGBA, cx, cy, r int) {
	drawDisc(dst, cx, cy, r, colPlusBg)
	// "+" 十字
	thick := r / 4
	armLen := r * 10 / 17
	for y := cy - thick/2; y <= cy+thick/2; y++ {
		for x := cx - armLen; x <= cx+armLen; x++ {
			if x >= 0 && y >= 0 && x < dst.Bounds().Dx() && y < dst.Bounds().Dy() {
				dst.SetRGBA(x, y, colPlusFg)
			}
		}
	}
	for x := cx - thick/2; x <= cx+thick/2; x++ {
		for y := cy - armLen; y <= cy+armLen; y++ {
			if x >= 0 && y >= 0 && x < dst.Bounds().Dx() && y < dst.Bounds().Dy() {
				dst.SetRGBA(x, y, colPlusFg)
			}
		}
	}
}
