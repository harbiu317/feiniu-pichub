package image

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
)

type Options struct {
	AutoCompress    bool
	Quality         int
	ConvertWebP     bool
	StripEXIF       bool
	WatermarkText   string
	WatermarkOpacity int
	ThumbSmall      int
	ThumbMedium     int
}

type Result struct {
	Key       string  // 最终存储 key
	ThumbSm   string
	ThumbMed  string
	Width     int
	Height    int
	Size      int64
	MIME      string
	Hash      string
	Data      []byte   // 最终数据
	SmallData []byte
	MedData   []byte
}

func randID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Process 处理上传的原始数据，返回持久化所需的 payload。
func Process(raw []byte, origName, mime string, opt Options) (*Result, error) {
	ext := strings.ToLower(filepath.Ext(origName))
	if ext == "" {
		ext = ".jpg"
	}

	isGIF := mime == "image/gif" || ext == ".gif"

	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("解码图片失败: %w", err)
	}

	b := img.Bounds()
	width, height := b.Dx(), b.Dy()

	// 水印
	if opt.WatermarkText != "" {
		img = drawWatermark(img, opt.WatermarkText, opt.WatermarkOpacity)
	}

	// 编码主图
	var finalBytes []byte
	var finalMIME, finalExt string

	switch {
	case isGIF:
		// GIF 保留动画；若 StripEXIF 则仅清理 comment / plaintext / non-loop application 扩展块
		if opt.StripEXIF {
			finalBytes = stripGIFMetadata(raw)
		} else {
			finalBytes = raw
		}
		finalMIME = "image/gif"
		finalExt = ".gif"
	case opt.ConvertWebP:
		// WebP 编码（调用 chai2010/webp）
		data, err := encodeWebP(img, opt.Quality)
		if err != nil {
			return nil, err
		}
		finalBytes = data
		finalMIME = "image/webp"
		finalExt = ".webp"
	default:
		out, m, e := encodeOriginal(img, ext, opt)
		if e != nil {
			return nil, e
		}
		finalBytes, finalMIME, finalExt = out, m, "."+fileExtFromMIME(m)
	}

	// 生成 key: 2026/04/15/<16char>.<ext>
	now := time.Now()
	id := randID(6)
	key := fmt.Sprintf("%04d/%02d/%02d/%s%s", now.Year(), now.Month(), now.Day(), id, finalExt)

	res := &Result{
		Key:    key,
		Width:  width,
		Height: height,
		Size:   int64(len(finalBytes)),
		MIME:   finalMIME,
		Data:   finalBytes,
		Hash:   sha256hex(finalBytes),
	}

	// 缩略图（GIF 也生成静态缩略图）
	if opt.ThumbSmall > 0 {
		sm, err := makeThumb(img, opt.ThumbSmall, opt.Quality)
		if err == nil {
			res.SmallData = sm
			res.ThumbSm = fmt.Sprintf("%04d/%02d/%02d/%s.s.jpg", now.Year(), now.Month(), now.Day(), id)
		}
	}
	if opt.ThumbMedium > 0 {
		md, err := makeThumb(img, opt.ThumbMedium, opt.Quality)
		if err == nil {
			res.MedData = md
			res.ThumbMed = fmt.Sprintf("%04d/%02d/%02d/%s.m.jpg", now.Year(), now.Month(), now.Day(), id)
		}
	}
	return res, nil
}

func encodeOriginal(img image.Image, ext string, opt Options) ([]byte, string, error) {
	var buf bytes.Buffer
	switch ext {
	case ".png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/png", nil
	case ".gif":
		if err := gif.Encode(&buf, img, nil); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/gif", nil
	default:
		q := opt.Quality
		if q <= 0 || q > 100 {
			q = 85
		}
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/jpeg", nil
	}
}

func makeThumb(src image.Image, max, quality int) ([]byte, error) {
	var dst image.Image
	b := src.Bounds()
	if b.Dx() > max || b.Dy() > max {
		dst = imaging.Fit(src, max, max, imaging.Lanczos)
	} else {
		dst = src
	}
	var buf bytes.Buffer
	q := quality
	if q <= 0 {
		q = 80
	}
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: q}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sha256hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func fileExtFromMIME(m string) string {
	switch m {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	}
	return "bin"
}

// ReadAll 带大小限制地读取上传流
func ReadAll(r io.Reader, maxBytes int64) ([]byte, error) {
	buf := &bytes.Buffer{}
	n, err := io.Copy(buf, io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if n > maxBytes {
		return nil, errors.New("文件超过大小限制")
	}
	return buf.Bytes(), nil
}
