package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/levis/pichub/internal/auth"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
		writeErr(w, http.StatusBadRequest, "参数错误")
		return
	}
	var (
		id       int64
		hash     string
		role     string
		disabled int
	)
	err := s.db.QueryRow(`SELECT id, password, role, disabled FROM users WHERE username = ?`, req.Username).
		Scan(&id, &hash, &role, &disabled)
	if err != nil || disabled == 1 || !auth.CheckPassword(hash, req.Password) {
		writeErr(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}
	token, err := s.jwt.Sign(id, req.Username, role)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "签发 token 失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":       id,
			"username": req.Username,
			"role":     role,
		},
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.FromContext(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "未登录")
		return
	}
	var email string
	var quotaMB int64
	var usedBytes int64
	_ = s.db.QueryRow(`SELECT email, quota_mb, used_bytes FROM users WHERE id = ?`, u.ID).
		Scan(&email, &quotaMB, &usedBytes)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":         u.ID,
		"username":   u.Username,
		"role":       u.Role,
		"email":      email,
		"quota_mb":   quotaMB,
		"used_bytes": usedBytes,
		"ts":         time.Now().Unix(),
	})
}
