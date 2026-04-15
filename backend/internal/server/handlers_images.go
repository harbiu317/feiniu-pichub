package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/levis/pichub/internal/auth"
)

func (s *Server) handleListImages(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.FromContext(r.Context())

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(q.Get("size"))
	if size < 1 || size > 200 {
		size = 48
	}
	keyword := strings.TrimSpace(q.Get("q"))
	albumID, _ := strconv.ParseInt(q.Get("album"), 10, 64)

	where := []string{"1=1"}
	args := []any{}

	// 普通用户只看自己的；管理员看全部（传 ?all=1）
	if u != nil {
		if !(u.Role == "admin" && q.Get("all") == "1") {
			where = append(where, "user_id = ?")
			args = append(args, u.ID)
		}
	} else {
		where = append(where, "user_id = 0")
	}

	if keyword != "" {
		where = append(where, "(filename LIKE ? OR tags LIKE ?)")
		args = append(args, "%"+keyword+"%", "%"+keyword+"%")
	}
	if albumID > 0 {
		where = append(where, "album_id = ?")
		args = append(args, albumID)
	}

	whereStr := strings.Join(where, " AND ")
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM images WHERE %s", whereStr)
	var total int64
	_ = s.db.QueryRow(countSQL, args...).Scan(&total)

	listSQL := fmt.Sprintf(`
		SELECT id, user_id, album_id, key, filename, mime, size, width, height, thumb_small, thumb_med, views, created_at
		FROM images WHERE %s ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, whereStr)
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(listSQL, args...)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var (
			id, uid, aid, created, views int64
			sz, width, height            int64
			key, fn, mime, thSm, thMd    string
		)
		_ = rows.Scan(&id, &uid, &aid, &key, &fn, &mime, &sz, &width, &height, &thSm, &thMd, &views, &created)
		url := s.store.URL(key)
		thumb := url
		if thMd != "" {
			thumb = s.store.URL(thMd)
		} else if thSm != "" {
			thumb = s.store.URL(thSm)
		}
		items = append(items, map[string]any{
			"id":         id,
			"user_id":    uid,
			"album_id":   aid,
			"key":        key,
			"filename":   fn,
			"mime":       mime,
			"size":       sz,
			"width":      width,
			"height":     height,
			"views":      views,
			"url":        url,
			"thumb":      thumb,
			"created_at": created,
		})
	}

	writeJSON(w, 200, map[string]any{
		"items": items,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

func (s *Server) handleGetImage(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var (
		uid, aid, created, views, sz, width, height int64
		key, fn, mime, thSm, thMd                   string
	)
	err := s.db.QueryRow(`SELECT user_id, album_id, key, filename, mime, size, width, height, thumb_small, thumb_med, views, created_at FROM images WHERE id = ?`, id).
		Scan(&uid, &aid, &key, &fn, &mime, &sz, &width, &height, &thSm, &thMd, &views, &created)
	if err != nil {
		writeErr(w, 404, "未找到")
		return
	}
	url := s.store.URL(key)
	thumb := url
	if thMd != "" {
		thumb = s.store.URL(thMd)
	}
	writeJSON(w, 200, map[string]any{
		"id": id, "user_id": uid, "album_id": aid, "key": key, "filename": fn, "mime": mime,
		"size": sz, "width": width, "height": height, "views": views, "url": url,
		"thumb":      thumb,
		"markdown":   fmt.Sprintf("![%s](%s)", fn, url),
		"html":       fmt.Sprintf(`<img src="%s" alt="%s" />`, url, fn),
		"bbcode":     fmt.Sprintf("[img]%s[/img]", url),
		"created_at": created,
	})
}

func (s *Server) handleDeleteImage(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.FromContext(r.Context())
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)

	var uid int64
	var key, thSm, thMd string
	var size int64
	err := s.db.QueryRow(`SELECT user_id, key, thumb_small, thumb_med, size FROM images WHERE id = ?`, id).
		Scan(&uid, &key, &thSm, &thMd, &size)
	if err != nil {
		writeErr(w, 404, "未找到")
		return
	}
	if u == nil || (u.Role != "admin" && u.ID != uid) {
		writeErr(w, 403, "无权限")
		return
	}

	_ = s.store.Delete(key)
	if thSm != "" {
		_ = s.store.Delete(thSm)
	}
	if thMd != "" {
		_ = s.store.Delete(thMd)
	}
	_, _ = s.db.Exec(`DELETE FROM images WHERE id = ?`, id)
	if uid > 0 {
		_, _ = s.db.Exec(`UPDATE users SET used_bytes = MAX(0, used_bytes - ?) WHERE id = ?`, size, uid)
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleBatchDelete(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.FromContext(r.Context())
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "参数错误")
		return
	}
	deleted := 0
	for _, id := range req.IDs {
		var uid int64
		var key, thSm, thMd string
		var size int64
		err := s.db.QueryRow(`SELECT user_id, key, thumb_small, thumb_med, size FROM images WHERE id = ?`, id).
			Scan(&uid, &key, &thSm, &thMd, &size)
		if err != nil {
			continue
		}
		if u.Role != "admin" && u.ID != uid {
			continue
		}
		_ = s.store.Delete(key)
		if thSm != "" {
			_ = s.store.Delete(thSm)
		}
		if thMd != "" {
			_ = s.store.Delete(thMd)
		}
		_, _ = s.db.Exec(`DELETE FROM images WHERE id = ?`, id)
		if uid > 0 {
			_, _ = s.db.Exec(`UPDATE users SET used_bytes = MAX(0, used_bytes - ?) WHERE id = ?`, size, uid)
		}
		deleted++
	}
	writeJSON(w, 200, map[string]any{"deleted": deleted})
}
