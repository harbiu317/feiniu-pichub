package server

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/levis/pichub/internal/auth"
	"github.com/levis/pichub/internal/config"
	"github.com/levis/pichub/internal/db"
	pichubImage "github.com/levis/pichub/internal/image"
	"github.com/levis/pichub/internal/storage"
)

//go:embed all:assets
var embeddedAssets embed.FS

// Version 由 main 包通过 ldflags 注入，对外暴露给 /api/meta。
var Version = "dev"

type Server struct {
	cfg   *config.Config
	http  *http.Server
	mux   *http.ServeMux
	db    *sql.DB
	jwt   *auth.JWT
	mw    *auth.Middleware
	store storage.Storage
	rate  *rateLimiter
}

func New(cfg *config.Config) (*Server, error) {
	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		return nil, err
	}
	if err := auth.EnsureAdmin(database.DB, cfg.Auth.AdminUser, cfg.Auth.AdminPassword); err != nil {
		return nil, err
	}

	jwtMgr := auth.NewJWT(cfg.Auth.JWTSecret, cfg.Auth.TokenLifetime)

	s := &Server{
		cfg:  cfg,
		db:   database.DB,
		jwt:  jwtMgr,
		mw:   &auth.Middleware{JWT: jwtMgr, DB: database.DB},
		rate: newRateLimiter(cfg.Limits.UploadPerMinute),
		mux:  http.NewServeMux(),
	}
	// 先用 DB settings 覆盖 cfg，再构建 storage，否则重启后 s.store 仍是 YAML 里的 driver
	s.applyDynamicSettings()
	store, err := buildStorage(cfg)
	if err != nil {
		return nil, err
	}
	s.store = store
	pichubImage.SetWatermarkFontPath(cfg.Image.Watermark.Font)
	s.routes()

	s.http = &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           s.withLogging(s.withCORS(s.mw.Wrap(s.mux))),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return s, nil
}

func (s *Server) Start() error {
	err := s.http.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	_ = s.db.Close()
	return s.http.Shutdown(ctx)
}

func (s *Server) routes() {
	// 公共
	s.mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"status": "ok"})
	})
	s.mux.HandleFunc("GET /api/meta", func(w http.ResponseWriter, r *http.Request) {
		driver := s.cfg.Storage.Driver
		if driver == "" {
			driver = "local"
		}
		writeJSON(w, 200, map[string]any{
			"version":        Version,
			"storage_driver": driver,
		})
	})
	s.mux.HandleFunc("GET /api/setup/status", s.handleSetupStatus)
	s.mux.HandleFunc("POST /api/setup/init", s.handleSetupInit)
	s.mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	s.mux.HandleFunc("GET /api/auth/me", s.handleMe)
	s.mux.HandleFunc("GET /api/stats", s.handleStats)

	// 上传（匿名或登录）
	s.mux.HandleFunc("POST /api/upload", s.limitUpload(s.handleUpload))

	// 图片
	s.mux.HandleFunc("GET /api/images", s.handleListImages)
	s.mux.HandleFunc("GET /api/images/{id}", s.handleGetImage)
	s.mux.HandleFunc("DELETE /api/images/{id}", auth.Require(s.handleDeleteImage))
	s.mux.HandleFunc("POST /api/images/batch-delete", auth.Require(s.handleBatchDelete))
	s.mux.HandleFunc("POST /api/images/move", auth.Require(s.handleMoveToAlbum))

	// 相册
	s.mux.HandleFunc("GET /api/albums", auth.Require(s.handleListAlbums))
	s.mux.HandleFunc("POST /api/albums", auth.Require(s.handleCreateAlbum))
	s.mux.HandleFunc("PUT /api/albums/{id}", auth.Require(s.handleUpdateAlbum))
	s.mux.HandleFunc("DELETE /api/albums/{id}", auth.Require(s.handleDeleteAlbum))

	// API Token
	s.mux.HandleFunc("GET /api/tokens", auth.Require(s.handleListTokens))
	s.mux.HandleFunc("POST /api/tokens", auth.Require(s.handleCreateToken))
	s.mux.HandleFunc("DELETE /api/tokens/{id}", auth.Require(s.handleDeleteToken))

	// 管理员
	s.mux.HandleFunc("GET /api/admin/users", auth.RequireAdmin(s.handleAdminListUsers))
	s.mux.HandleFunc("POST /api/admin/users", auth.RequireAdmin(s.handleAdminCreateUser))
	s.mux.HandleFunc("PUT /api/admin/users/{id}", auth.RequireAdmin(s.handleAdminUpdateUser))
	s.mux.HandleFunc("DELETE /api/admin/users/{id}", auth.RequireAdmin(s.handleAdminDeleteUser))
	s.mux.HandleFunc("GET /api/admin/settings", auth.RequireAdmin(s.handleAdminSettings))
	s.mux.HandleFunc("PUT /api/admin/settings", auth.RequireAdmin(s.handleAdminSettings))
	s.mux.HandleFunc("POST /api/admin/storage/test", auth.RequireAdmin(s.handleStorageTest))

	// 图片出图
	s.mux.HandleFunc("GET /i/", s.handleServeImage)
	s.mux.HandleFunc("GET /t/{id}", s.handleThumbnail)

	// 前端（SPA）
	s.mux.HandleFunc("GET /", s.handleSPA)
}

func (s *Server) handleThumbnail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var thumb, key string
	_ = s.db.QueryRow(`SELECT thumb_med, key FROM images WHERE id = ?`, id).Scan(&thumb, &key)
	target := thumb
	if target == "" {
		target = key
	}
	if target == "" {
		http.NotFound(w, r)
		return
	}
	r.URL.Path = "/i/" + target
	s.handleServeImage(w, r)
}

func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	assets, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		http.Error(w, "前端资源未嵌入", 500)
		return
	}
	p := strings.TrimPrefix(r.URL.Path, "/")
	if p == "" {
		p = "index.html"
	}
	if _, err := fs.Stat(assets, p); err != nil {
		// SPA 路由回退
		data, err := fs.ReadFile(assets, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
		return
	}
	http.FileServer(http.FS(assets)).ServeHTTP(w, r)
}

// helpers
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Token")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sr := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(sr, r)
		if !strings.HasPrefix(r.URL.Path, "/i/") && !strings.HasPrefix(r.URL.Path, "/assets/") {
			// 简洁日志
			println(time.Now().Format("15:04:05"), r.Method, r.URL.Path, sr.status, time.Since(start).String())
		}
	})
}
