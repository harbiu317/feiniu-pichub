package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/levis/pichub/internal/auth"
)

func (s *Server) needsSetup() bool {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&count)
	return count == 0
}

func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"needs_setup": s.needsSetup(),
	})
}

func (s *Server) handleSetupInit(w http.ResponseWriter, r *http.Request) {
	// 只在没有管理员时允许，防止被滥用
	if !s.needsSetup() {
		writeErr(w, http.StatusForbidden, "已完成初始化，不可重复设置")
		return
	}
	var req struct {
		Username       string `json:"username"`
		Password       string `json:"password"`
		AllowAnonymous *bool  `json:"allow_anonymous"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "参数错误")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) < 2 {
		writeErr(w, 400, "用户名长度至少 2 位")
		return
	}
	if len(req.Password) < 6 {
		writeErr(w, 400, "密码长度至少 6 位")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	now := time.Now().Unix()
	_, err = s.db.Exec(`INSERT INTO users (username, password, role, created_at, updated_at) VALUES (?, ?, 'admin', ?, ?)`,
		req.Username, hash, now, now)
	if err != nil {
		writeErr(w, 500, "创建失败: "+err.Error())
		return
	}
	if req.AllowAnonymous != nil {
		s.cfg.Server.AllowAnon = *req.AllowAnonymous
		b, _ := json.Marshal(*req.AllowAnonymous)
		_, _ = s.db.Exec(`INSERT INTO settings(key, value) VALUES('allow_anonymous', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, string(b))
	}
	token, _ := s.jwt.Sign(1, req.Username, "admin")
	writeJSON(w, 200, map[string]any{
		"ok":    true,
		"token": token,
		"user":  map[string]any{"id": 1, "username": req.Username, "role": "admin"},
	})
}
