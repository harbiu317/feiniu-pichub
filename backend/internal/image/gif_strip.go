package image

import (
	"bytes"
	"errors"
)

// stripGIFMetadata 在保留动画和帧数据的前提下，移除 GIF 中的元数据扩展块：
//   - Comment Extension (0x21 0xFE)
//   - Plain Text Extension (0x21 0x01)
//   - Application Extension (0x21 0xFF)，但保留 NETSCAPE2.0（循环控制）和 XMP Data（用户可选）
//
// 若遇到无法解析的结构，原样返回避免破坏文件。
func stripGIFMetadata(raw []byte) []byte {
	out, err := stripGIFMetadataErr(raw)
	if err != nil {
		return raw
	}
	return out
}

func stripGIFMetadataErr(raw []byte) ([]byte, error) {
	if len(raw) < 13 {
		return nil, errors.New("not a gif")
	}
	// Header: "GIF87a" or "GIF89a"
	if !bytes.Equal(raw[:3], []byte("GIF")) {
		return nil, errors.New("not a gif")
	}

	var buf bytes.Buffer
	// 写 Header (6) + LSD (7) = 13 字节
	buf.Write(raw[:13])
	packed := raw[10]
	globalCT := packed&0x80 != 0
	gctSize := 3 * (1 << ((packed & 0x07) + 1))
	pos := 13
	if globalCT {
		if pos+gctSize > len(raw) {
			return nil, errors.New("truncated gct")
		}
		buf.Write(raw[pos : pos+gctSize])
		pos += gctSize
	}

	for pos < len(raw) {
		b := raw[pos]
		switch b {
		case 0x3B: // Trailer
			buf.WriteByte(0x3B)
			return buf.Bytes(), nil
		case 0x2C: // Image Descriptor (+ image data)
			// 10 bytes header
			if pos+10 > len(raw) {
				return nil, errors.New("truncated image desc")
			}
			buf.Write(raw[pos : pos+10])
			pack := raw[pos+9]
			pos += 10
			if pack&0x80 != 0 {
				// local color table
				sz := 3 * (1 << ((pack & 0x07) + 1))
				if pos+sz > len(raw) {
					return nil, errors.New("truncated lct")
				}
				buf.Write(raw[pos : pos+sz])
				pos += sz
			}
			// LZW minimum code size
			if pos >= len(raw) {
				return nil, errors.New("truncated lzw")
			}
			buf.WriteByte(raw[pos])
			pos++
			// sub-blocks
			next, err := copySubBlocks(&buf, raw, pos)
			if err != nil {
				return nil, err
			}
			pos = next
		case 0x21: // Extension
			if pos+2 > len(raw) {
				return nil, errors.New("truncated ext")
			}
			label := raw[pos+1]
			switch label {
			case 0xF9:
				// Graphic Control — 必须保留（动画控制）
				buf.Write(raw[pos : pos+2])
				pos += 2
				next, err := copySubBlocks(&buf, raw, pos)
				if err != nil {
					return nil, err
				}
				pos = next
			case 0xFF:
				// Application Extension — 保留 NETSCAPE2.0/ANIMEXTS1.0 用于循环
				if pos+3 > len(raw) {
					return nil, errors.New("truncated app ext")
				}
				sz := int(raw[pos+2])
				if pos+3+sz > len(raw) {
					return nil, errors.New("truncated app ext body")
				}
				app := string(raw[pos+3 : pos+3+sz])
				keep := app == "NETSCAPE2.0" || app == "ANIMEXTS1.0"
				if keep {
					buf.Write(raw[pos : pos+3+sz])
					pos += 3 + sz
					next, err := copySubBlocks(&buf, raw, pos)
					if err != nil {
						return nil, err
					}
					pos = next
				} else {
					// skip header + body sub-blocks
					pos += 3 + sz
					next, err := skipSubBlocks(raw, pos)
					if err != nil {
						return nil, err
					}
					pos = next
				}
			case 0xFE, 0x01:
				// Comment (0xFE) / Plain Text (0x01) — 丢弃
				pos += 2
				if label == 0x01 {
					// Plain Text has 12-byte header sub-block
					if pos+13 > len(raw) {
						return nil, errors.New("truncated plaintext header")
					}
					pos += 13
				}
				next, err := skipSubBlocks(raw, pos)
				if err != nil {
					return nil, err
				}
				pos = next
			default:
				// 未知扩展：安全起见保留
				buf.Write(raw[pos : pos+2])
				pos += 2
				next, err := copySubBlocks(&buf, raw, pos)
				if err != nil {
					return nil, err
				}
				pos = next
			}
		default:
			return nil, errors.New("unknown block")
		}
	}
	return buf.Bytes(), nil
}

func copySubBlocks(buf *bytes.Buffer, raw []byte, pos int) (int, error) {
	for {
		if pos >= len(raw) {
			return 0, errors.New("truncated sub-blocks")
		}
		sz := int(raw[pos])
		buf.WriteByte(byte(sz))
		pos++
		if sz == 0 {
			return pos, nil
		}
		if pos+sz > len(raw) {
			return 0, errors.New("truncated sub-block body")
		}
		buf.Write(raw[pos : pos+sz])
		pos += sz
	}
}

func skipSubBlocks(raw []byte, pos int) (int, error) {
	for {
		if pos >= len(raw) {
			return 0, errors.New("truncated sub-blocks")
		}
		sz := int(raw[pos])
		pos++
		if sz == 0 {
			return pos, nil
		}
		if pos+sz > len(raw) {
			return 0, errors.New("truncated sub-block body")
		}
		pos += sz
	}
}
