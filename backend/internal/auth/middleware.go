package auth

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
)

type ctxKey string

const UserCtxKey ctxKey = "pichub.user"

type User struct {
	ID       int64
	Username string
	Role     string
}

func FromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(UserCtxKey).(*User)
	return u, ok
}

// Middleware 解析 Authorization: Bearer <jwt> 或 X-Token: <api token>。
// 未登录时不拒绝（具体接口决定是否要求鉴权）。
type Middleware struct {
	JWT *JWT
	DB  *sql.DB
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u := m.resolve(r); u != nil {
			r = r.WithContext(context.WithValue(r.Context(), UserCtxKey, u))
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) resolve(r *http.Request) *User {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		if c, err := m.JWT.Parse(strings.TrimPrefix(h, "Bearer ")); err == nil {
			return &User{ID: c.UserID, Username: c.Username, Role: c.Role}
		}
	}
	tok := r.Header.Get("X-Token")
	if tok == "" {
		tok = r.URL.Query().Get("token")
	}
	if tok == "" {
		if c, err := r.Cookie("pichub_token"); err == nil {
			tok = c.Value
		}
	}
	if tok == "" {
		return nil
	}
	var uid int64
	var username, role string
	err := m.DB.QueryRow(`
		SELECT u.id, u.username, u.role FROM tokens t
		JOIN users u ON u.id = t.user_id
		WHERE t.token = ? AND u.disabled = 0
		  AND (t.expires_at = 0 OR t.expires_at > strftime('%s','now'))
	`, tok).Scan(&uid, &username, &role)
	if err != nil {
		return nil
	}
	_, _ = m.DB.Exec(`UPDATE tokens SET last_used = strftime('%s','now') WHERE token = ?`, tok)
	return &User{ID: uid, Username: username, Role: role}
}

// Require 强制要求已登录用户。
func Require(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := FromContext(r.Context()); !ok {
			http.Error(w, `{"error":"未登录"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := FromContext(r.Context())
		if !ok || u.Role != "admin" {
			http.Error(w, `{"error":"需要管理员权限"}`, http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
