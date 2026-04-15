package server

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/levis/pichub/internal/auth"
	"github.com/levis/pichub/internal/image"
)

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	// 匿名上传开关
	user, _ := auth.FromContext(r.Context())
	if user == nil && !s.cfg.Server.AllowAnon {
		writeErr(w, http.StatusUnauthorized, "当前站点不允许匿名上传")
		return
	}

	maxBytes := int64(s.cfg.Image.MaxSizeMB) * 1024 * 1024
	if err := r.ParseMultipartForm(maxBytes + 4*1024*1024); err != nil {
		writeErr(w, http.StatusBadRequest, "解析上传失败: "+err.Error())
		return
	}

	form := r.MultipartForm
	files := form.File["file"]
	if len(files) == 0 {
		files = form.File["files"]
	}
	if len(files) == 0 {
		writeErr(w, http.StatusBadRequest, "未上传文件")
		return
	}

	opt := image.Options{
		AutoCompress:     s.cfg.Image.AutoCompress,
		Quality:          s.cfg.Image.Quality,
		ConvertWebP:      s.cfg.Image.ConvertWebP,
		StripEXIF:        s.cfg.Image.StripEXIF,
		ThumbSmall:       s.cfg.Image.Thumbnail.Small,
		ThumbMedium:      s.cfg.Image.Thumbnail.Medium,
	}
	if s.cfg.Image.Watermark.Enabled {
		opt.WatermarkText = s.cfg.Image.Watermark.Text
		opt.WatermarkOpacity = s.cfg.Image.Watermark.Opacity
	}

	results := []map[string]any{}
	errs := []string{}

	// 配额预检：拉取用户当前 quota / used
	var quotaMB, usedBytes int64
	if user != nil {
		_ = s.db.QueryRow(`SELECT quota_mb, used_bytes FROM users WHERE id = ?`, user.ID).
			Scan(&quotaMB, &usedBytes)
	}
	// 本次上传已消耗的字节（用于单次请求内多文件的累计判断）
	var consumed int64

	for _, fh := range files {
		if fh.Size > maxBytes {
			errs = append(errs, fh.Filename+": 超过大小限制")
			continue
		}
		ct := fh.Header.Get("Content-Type")
		if !isAllowed(ct, s.cfg.Image.AllowedTypes) {
			errs = append(errs, fh.Filename+": 不支持的类型 "+ct)
			continue
		}
		f, err := fh.Open()
		if err != nil {
			errs = append(errs, fh.Filename+": "+err.Error())
			continue
		}
		raw, err := image.ReadAll(f, maxBytes)
		f.Close()
		if err != nil {
			errs = append(errs, fh.Filename+": "+err.Error())
			continue
		}
		res, err := image.Process(raw, fh.Filename, ct, opt)
		if err != nil {
			errs = append(errs, fh.Filename+": "+err.Error())
			continue
		}

		// 配额校验（仅登录用户且设置了配额时）
		if user != nil && quotaMB > 0 {
			total := res.Size + int64(len(res.SmallData)) + int64(len(res.MedData))
			if usedBytes+consumed+total > quotaMB*1024*1024 {
				remaining := quotaMB*1024*1024 - usedBytes - consumed
				if remaining < 0 {
					remaining = 0
				}
				errs = append(errs, fmt.Sprintf("%s: 超出配额（剩余 %s / 总 %d MB）", fh.Filename, formatBytes(remaining), quotaMB))
				continue
			}
			consumed += total
		}

		// 写入存储
		if err := s.store.Put(res.Key, bytes.NewReader(res.Data), res.Size, res.MIME); err != nil {
			errs = append(errs, fh.Filename+": 存储失败 "+err.Error())
			continue
		}
		if res.ThumbSm != "" {
			_ = s.store.Put(res.ThumbSm, bytes.NewReader(res.SmallData), int64(len(res.SmallData)), "image/jpeg")
		}
		if res.ThumbMed != "" {
			_ = s.store.Put(res.ThumbMed, bytes.NewReader(res.MedData), int64(len(res.MedData)), "image/jpeg")
		}

		// 入库
		var uid int64
		if user != nil {
			uid = user.ID
		}
		ip := clientIP(r, s.cfg.Server.TrustProxy)
		now := time.Now().Unix()
		_, err = s.db.Exec(`
			INSERT INTO images (user_id, key, filename, mime, size, width, height, hash, thumb_small, thumb_med, ip, created_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
		`, uid, res.Key, fh.Filename, res.MIME, res.Size, res.Width, res.Height, res.Hash, res.ThumbSm, res.ThumbMed, ip, now)
		if err != nil {
			errs = append(errs, fh.Filename+": 数据库写入失败 "+err.Error())
			continue
		}
		if uid > 0 {
			// 记录主图 + 缩略图的总占用
			total := res.Size + int64(len(res.SmallData)) + int64(len(res.MedData))
			_, _ = s.db.Exec(`UPDATE users SET used_bytes = used_bytes + ? WHERE id = ?`, total, uid)
		}

		url := s.store.URL(res.Key)
		thumbURL := ""
		if res.ThumbMed != "" {
			thumbURL = s.store.URL(res.ThumbMed)
		}
		results = append(results, map[string]any{
			"key":       res.Key,
			"filename":  fh.Filename,
			"ext":       strings.TrimPrefix(filepath.Ext(res.Key), "."),
			"mime":      res.MIME,
			"size":      res.Size,
			"width":     res.Width,
			"height":    res.Height,
			"url":       url,
			"thumb":     thumbURL,
			"markdown":  fmt.Sprintf("![%s](%s)", fh.Filename, url),
			"html":      fmt.Sprintf(`<img src="%s" alt="%s" />`, url, fh.Filename),
			"bbcode":    fmt.Sprintf("[img]%s[/img]", url),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"uploaded": results,
		"errors":   errs,
	})
}

func formatBytes(b int64) string {
	switch {
	case b < 1024:
		return fmt.Sprintf("%d B", b)
	case b < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(b)/1024/1024)
	}
}

func isAllowed(ct string, list []string) bool {
	if len(list) == 0 {
		return true
	}
	for _, a := range list {
		if strings.EqualFold(a, ct) {
			return true
		}
	}
	return false
}

func clientIP(r *http.Request, trust bool) string {
	if trust {
		if v := r.Header.Get("X-Forwarded-For"); v != "" {
			parts := strings.Split(v, ",")
			return strings.TrimSpace(parts[0])
		}
		if v := r.Header.Get("X-Real-IP"); v != "" {
			return v
		}
	}
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx > 0 {
		ip = ip[:idx]
	}
	return ip
}
