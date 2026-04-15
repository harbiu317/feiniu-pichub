package auth

import (
	"database/sql"
	"log"
	"time"
)

// EnsureAdmin 按提供的凭证创建或重置管理员。
// 如果 DB 中已有同名管理员：更新其密码（wizard 重新运行 / 忘记密码场景）。
// 如果 DB 中没有任何管理员：创建一个。
// 如果 DB 中有其它名字的管理员且提供了新用户名：额外创建新管理员（不删除旧的）。
// username / password 任一为空则跳过。
func EnsureAdmin(db *sql.DB, username, password string) error {
	if username == "" || password == "" {
		return nil
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	now := time.Now().Unix()

	var existingID int64
	err = db.QueryRow(`SELECT id FROM users WHERE username = ?`, username).Scan(&existingID)
	if err == sql.ErrNoRows {
		// 用户名不存在 → 新建管理员
		_, err = db.Exec(`
			INSERT INTO users (username, password, role, created_at, updated_at)
			VALUES (?, ?, 'admin', ?, ?)
		`, username, hash, now, now)
		if err == nil {
			log.Printf("[auth] 创建管理员账户: %s", username)
		}
		return err
	}
	if err != nil {
		return err
	}
	// 用户名已存在 → 重置密码并升级为 admin
	_, err = db.Exec(`UPDATE users SET password = ?, role = 'admin', disabled = 0, updated_at = ? WHERE id = ?`,
		hash, now, existingID)
	if err == nil {
		log.Printf("[auth] 更新管理员 %s 密码", username)
	}
	return err
}
