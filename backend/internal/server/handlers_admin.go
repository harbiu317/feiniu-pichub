package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/levis/pichub/internal/auth"
)

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`SELECT id, username, email, role, quota_mb, used_bytes, disabled, created_at FROM users ORDER BY id ASC`)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, quota, used, created int64
		var disabled int
		var username, email, role string
		_ = rows.Scan(&id, &username, &email, &role, &quota, &used, &disabled, &created)
		out = append(out, map[string]any{
			"id": id, "username": username, "email": email, "role": role,
			"quota_mb": quota, "used_bytes": used, "disabled": disabled == 1, "created_at": created,
		})
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
		QuotaMB  int64  `json:"quota_mb"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		writeErr(w, 400, "缺少用户名或密码")
		return
	}
	if req.Role == "" {
		req.Role = "user"
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	now := time.Now().Unix()
	_, err = s.db.Exec(`INSERT INTO users (username, password, role, quota_mb, created_at, updated_at) VALUES (?,?,?,?,?,?)`,
		req.Username, hash, req.Role, req.QuotaMB, now, now)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var req struct {
		Role     string `json:"role"`
		QuotaMB  int64  `json:"quota_mb"`
		Disabled bool   `json:"disabled"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "参数错误")
		return
	}
	disabled := 0
	if req.Disabled {
		disabled = 1
	}
	_, err := s.db.Exec(`UPDATE users SET role=COALESCE(NULLIF(?, ''), role), quota_mb=?, disabled=?, updated_at=? WHERE id=?`,
		req.Role, req.QuotaMB, disabled, time.Now().Unix(), id)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if req.Password != "" {
		hash, _ := auth.HashPassword(req.Password)
		_, _ = s.db.Exec(`UPDATE users SET password=? WHERE id=?`, hash, id)
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id == 1 {
		writeErr(w, 400, "不能删除 #1 管理员")
		return
	}
	_, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleAdminSettings(w http.ResponseWriter, r *http.Request) {
	// GET 返回当前运行时配置；PUT 更新 settings 表（会被 runtime 合并覆盖）
	if r.Method == http.MethodGet {
		writeJSON(w, 200, map[string]any{
			"allow_anonymous":   s.cfg.Server.AllowAnon,
			"max_size_mb":       s.cfg.Image.MaxSizeMB,
			"allowed_types":     s.cfg.Image.AllowedTypes,
			"auto_compress":     s.cfg.Image.AutoCompress,
			"quality":           s.cfg.Image.Quality,
			"convert_webp":      s.cfg.Image.ConvertWebP,
			"strip_exif":        s.cfg.Image.StripEXIF,
			"watermark_enabled": s.cfg.Image.Watermark.Enabled,
			"watermark_text":    s.cfg.Image.Watermark.Text,
			"upload_per_minute": s.cfg.Limits.UploadPerMinute,
			// 存储
			"storage_driver":      s.cfg.Storage.Driver,
			"storage_local_root":  s.cfg.Storage.Local.Root,
			"storage_s3_endpoint": s.cfg.Storage.S3.Endpoint,
			"storage_s3_region":   s.cfg.Storage.S3.Region,
			"storage_s3_bucket":   s.cfg.Storage.S3.Bucket,
			"storage_s3_access":   s.cfg.Storage.S3.AccessKey,
			"storage_s3_secret":   maskSecret(s.cfg.Storage.S3.SecretKey),
			"storage_s3_ssl":      s.cfg.Storage.S3.UseSSL,
			"storage_s3_path":     s.cfg.Storage.S3.PathStyle,
			"storage_s3_public":   s.cfg.Storage.S3.PublicBase,
			"storage_s3_prefix":   s.cfg.Storage.S3.Prefix,
		})
		return
	}
	// PUT
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "参数错误")
		return
	}
	for k, v := range req {
		b, _ := json.Marshal(v)
		_, _ = s.db.Exec(`INSERT INTO settings(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, k, string(b))
	}
	s.applyDynamicSettings()
	// 存储驱动切换后重建 storage 实例
	if newStore, err := buildStorage(s.cfg); err == nil {
		s.store = newStore
	} else {
		writeJSON(w, 200, map[string]any{"ok": true, "warn": "存储配置未生效: " + err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func maskSecret(v string) string {
	if len(v) <= 6 {
		return ""
	}
	return v[:3] + "••••••" + v[len(v)-3:]
}

// applyDynamicSettings 从 settings 表覆盖内存配置
func (s *Server) applyDynamicSettings() {
	rows, err := s.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		_ = rows.Scan(&k, &v)
		var raw any
		_ = json.Unmarshal([]byte(v), &raw)
		asStr := func() string {
			if str, ok := raw.(string); ok {
				return str
			}
			return ""
		}
		asBool := func() bool {
			b, _ := raw.(bool)
			return b
		}
		asInt := func() int {
			f, _ := raw.(float64)
			return int(f)
		}
		switch k {
		case "allow_anonymous":
			s.cfg.Server.AllowAnon = asBool()
		case "max_size_mb":
			s.cfg.Image.MaxSizeMB = asInt()
		case "auto_compress":
			s.cfg.Image.AutoCompress = asBool()
		case "quality":
			s.cfg.Image.Quality = asInt()
		case "convert_webp":
			s.cfg.Image.ConvertWebP = asBool()
		case "strip_exif":
			s.cfg.Image.StripEXIF = asBool()
		case "watermark_enabled":
			s.cfg.Image.Watermark.Enabled = asBool()
		case "watermark_text":
			s.cfg.Image.Watermark.Text = asStr()
		case "storage_driver":
			s.cfg.Storage.Driver = asStr()
		case "storage_local_root":
			if str := asStr(); str != "" {
				s.cfg.Storage.Local.Root = str
			}
		case "storage_s3_endpoint":
			s.cfg.Storage.S3.Endpoint = asStr()
		case "storage_s3_region":
			s.cfg.Storage.S3.Region = asStr()
		case "storage_s3_bucket":
			s.cfg.Storage.S3.Bucket = asStr()
		case "storage_s3_access":
			s.cfg.Storage.S3.AccessKey = asStr()
		case "storage_s3_secret":
			// 只有非掩码值才更新
			str := asStr()
			if str != "" && !isMasked(str) {
				s.cfg.Storage.S3.SecretKey = str
			}
		case "storage_s3_ssl":
			s.cfg.Storage.S3.UseSSL = asBool()
		case "storage_s3_path":
			s.cfg.Storage.S3.PathStyle = asBool()
		case "storage_s3_public":
			s.cfg.Storage.S3.PublicBase = asStr()
		case "storage_s3_prefix":
			s.cfg.Storage.S3.Prefix = asStr()
		case "thumbnail":
			// thumbnail 是布尔值，控制是否生成缩略图
			s.cfg.Image.Thumbnail.Small = 200  // 保持默认值
			s.cfg.Image.Thumbnail.Medium = 600 // 保持默认值
		}
	}
}

func isMasked(s string) bool {
	for _, r := range s {
		if r == '•' {
			return true
		}
	}
	return false
}
