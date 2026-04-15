package server

import (
	"net/http"
	"time"

	"github.com/levis/pichub/internal/auth"
)

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.FromContext(r.Context())
	where := "1=1"
	args := []any{}
	if u == nil || u.Role != "admin" {
		if u != nil {
			where = "user_id = ?"
			args = append(args, u.ID)
		} else {
			where = "user_id = 0"
		}
	}

	var total, size int64
	_ = s.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(size),0) FROM images WHERE "+where, args...).Scan(&total, &size)

	startOfDay := time.Now().Truncate(24 * time.Hour).Unix()
	var today int64
	todaySQL := "SELECT COUNT(*) FROM images WHERE " + where + " AND created_at >= ?"
	_ = s.db.QueryRow(todaySQL, append(args, startOfDay)...).Scan(&today)

	var views int64
	_ = s.db.QueryRow("SELECT COALESCE(SUM(views),0) FROM images WHERE "+where, args...).Scan(&views)

	var albumCount int64
	if u != nil {
		_ = s.db.QueryRow("SELECT COUNT(*) FROM albums WHERE user_id = ?", u.ID).Scan(&albumCount)
	}

	writeJSON(w, 200, map[string]any{
		"total":        total,
		"size_bytes":   size,
		"today":        today,
		"views":        views,
		"album_count":  albumCount,
	})
}
