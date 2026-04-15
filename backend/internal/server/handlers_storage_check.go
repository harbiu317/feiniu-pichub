package server

import (
	"encoding/json"
	"net/http"

	"github.com/levis/pichub/internal/config"
	"github.com/levis/pichub/internal/storage"
)

// handleStorageTest 在不修改运行时配置的情况下，用传入的 S3 凭证做一次 HeadBucket 试连。
// 掩码 secret (•••) 时复用当前已保存的密钥。
func (s *Server) handleStorageTest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Driver    string `json:"driver"`
		Endpoint  string `json:"endpoint"`
		Region    string `json:"region"`
		Bucket    string `json:"bucket"`
		AccessKey string `json:"access_key"`
		SecretKey string `json:"secret_key"`
		UseSSL    bool   `json:"use_ssl"`
		PathStyle bool   `json:"path_style"`
		Prefix    string `json:"prefix"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "参数错误")
		return
	}
	if req.SecretKey == "" || isMasked(req.SecretKey) {
		req.SecretKey = s.cfg.Storage.S3.SecretKey
	}
	cfg := config.S3Config{
		Endpoint:  req.Endpoint,
		Region:    req.Region,
		Bucket:    req.Bucket,
		AccessKey: req.AccessKey,
		SecretKey: req.SecretKey,
		UseSSL:    req.UseSSL,
		PathStyle: req.PathStyle,
		Prefix:    req.Prefix,
	}
	driver, err := storage.NewS3(cfg)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	if err := driver.TestConnection(); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "message": "连接成功，bucket 可读写"})
}
