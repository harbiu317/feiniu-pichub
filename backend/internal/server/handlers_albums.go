package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/levis/pichub/internal/auth"
)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "album-" + strconv.FormatInt(time.Now().Unix(), 10)
	}
	return string(out)
}

func (s *Server) handleListAlbums(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.FromContext(r.Context())
	rows, err := s.db.Query(`
		SELECT a.id, a.name, a.slug, a.description, a.is_public, a.cover_key, a.created_at,
		       (SELECT COUNT(*) FROM images WHERE album_id = a.id) AS image_count
		FROM albums a WHERE a.user_id = ? ORDER BY a.created_at DESC
	`, u.ID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, created, count int64
		var pub int
		var name, slug, desc, cover string
		_ = rows.Scan(&id, &name, &slug, &desc, &pub, &cover, &created, &count)
		out = append(out, map[string]any{
			"id": id, "name": name, "slug": slug, "description": desc,
			"is_public": pub == 1, "cover_key": cover, "created_at": created,
			"image_count": count,
		})
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleCreateAlbum(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.FromContext(r.Context())
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsPublic    bool   `json:"is_public"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeErr(w, 400, "缺少相册名")
		return
	}
	now := time.Now().Unix()
	pub := 0
	if req.IsPublic {
		pub = 1
	}
	res, err := s.db.Exec(`INSERT INTO albums (user_id, name, slug, description, is_public, created_at, updated_at) VALUES (?,?,?,?,?,?,?)`,
		u.ID, req.Name, slugify(req.Name), req.Description, pub, now, now)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, 200, map[string]any{"id": id, "name": req.Name})
}

func (s *Server) handleUpdateAlbum(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.FromContext(r.Context())
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsPublic    bool   `json:"is_public"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "参数错误")
		return
	}
	pub := 0
	if req.IsPublic {
		pub = 1
	}
	_, err := s.db.Exec(`UPDATE albums SET name=?, description=?, is_public=?, updated_at=? WHERE id=? AND user_id=?`,
		req.Name, req.Description, pub, time.Now().Unix(), id, u.ID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleDeleteAlbum(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.FromContext(r.Context())
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	// 相册下的图片移到默认（album_id=0）
	_, _ = s.db.Exec(`UPDATE images SET album_id = 0 WHERE album_id = ? AND user_id = ?`, id, u.ID)
	_, err := s.db.Exec(`DELETE FROM albums WHERE id = ? AND user_id = ?`, id, u.ID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleMoveToAlbum(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.FromContext(r.Context())
	var req struct {
		ImageIDs []int64 `json:"image_ids"`
		AlbumID  int64   `json:"album_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "参数错误")
		return
	}
	if req.AlbumID > 0 {
		var cnt int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM albums WHERE id = ? AND user_id = ?`, req.AlbumID, u.ID).Scan(&cnt)
		if cnt == 0 {
			writeErr(w, 404, "相册不存在")
			return
		}
	}
	moved := 0
	for _, id := range req.ImageIDs {
		res, err := s.db.Exec(`UPDATE images SET album_id = ? WHERE id = ? AND user_id = ?`, req.AlbumID, id, u.ID)
		if err == nil {
			if n, _ := res.RowsAffected(); n > 0 {
				moved++
			}
		}
	}
	writeJSON(w, 200, map[string]any{"moved": moved})
}
