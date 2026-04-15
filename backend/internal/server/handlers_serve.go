package server

import (
	"io"
	"net/http"
	"strings"
)

// handleServeImage: GET /i/{year}/{month}/{day}/{name}
func (s *Server) handleServeImage(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/i/")
	if key == "" {
		http.NotFound(w, r)
		return
	}
	rc, size, err := s.store.Get(key)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer rc.Close()

	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Type", mimeFromKey(key))
	if size > 0 {
		w.Header().Set("Content-Length", itoa(size))
	}
	_, _ = io.Copy(w, rc)

	// 异步自增访问次数（忽略错误）
	go func() {
		_, _ = s.db.Exec(`UPDATE images SET views = views + 1 WHERE key = ? OR thumb_small = ? OR thumb_med = ?`, key, key, key)
	}()
}

func mimeFromKey(key string) string {
	ext := ""
	if i := strings.LastIndex(key, "."); i >= 0 {
		ext = strings.ToLower(key[i+1:])
	}
	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	}
	return "application/octet-stream"
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
