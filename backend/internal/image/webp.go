package image

import (
	"bytes"
	"image"

	nativewebp "github.com/HugoSmits86/nativewebp"
)

// encodeWebP 使用纯 Go 的 VP8L 无损编码（quality 参数保留以兼容调用方签名）。
func encodeWebP(img image.Image, _ int) ([]byte, error) {
	var buf bytes.Buffer
	if err := nativewebp.Encode(&buf, img, &nativewebp.Options{}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
