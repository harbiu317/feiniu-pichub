package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/levis/pichub/internal/auth"
)

func newAPIToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return "pk_" + base64.RawURLEncoding.EncodeToString(b)
}

func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.FromContext(r.Context())
	rows, err := s.db.Query(`SELECT id, name, token, last_used, created_at, expires_at FROM tokens WHERE user_id = ? ORDER BY created_at DESC`, u.ID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, last, created, exp int64
		var name, tok string
		_ = rows.Scan(&id, &name, &tok, &last, &created, &exp)
		out = append(out, map[string]any{
			"id":         id,
			"name":       name,
			"token":      tok,
			"last_used":  last,
			"created_at": created,
			"expires_at": exp,
		})
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.FromContext(r.Context())
	var req struct {
		Name    string `json:"name"`
		Days    int    `json:"days"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Name == "" {
		req.Name = "API Token"
	}
	tok := newAPIToken()
	var expires int64
	if req.Days > 0 {
		expires = time.Now().AddDate(0, 0, req.Days).Unix()
	}
	_, err := s.db.Exec(`INSERT INTO tokens (user_id, name, token, created_at, expires_at) VALUES (?, ?, ?, ?, ?)`,
		u.ID, req.Name, tok, time.Now().Unix(), expires)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"name": req.Name, "token": tok, "expires_at": expires})
}

func (s *Server) handleDeleteToken(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.FromContext(r.Context())
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	_, err := s.db.Exec(`DELETE FROM tokens WHERE id = ? AND user_id = ?`, id, u.ID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
