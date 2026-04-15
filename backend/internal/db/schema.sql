-- PicHub SQLite schema

CREATE TABLE IF NOT EXISTS users (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    username    TEXT NOT NULL UNIQUE,
    password    TEXT NOT NULL,
    email       TEXT DEFAULT '',
    role        TEXT NOT NULL DEFAULT 'user',  -- user | admin
    quota_mb    INTEGER NOT NULL DEFAULT 0,    -- 0 = 无限
    used_bytes  INTEGER NOT NULL DEFAULT 0,
    disabled    INTEGER NOT NULL DEFAULT 0,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS tokens (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL,
    name        TEXT NOT NULL,
    token       TEXT NOT NULL UNIQUE,
    last_used   INTEGER DEFAULT 0,
    created_at  INTEGER NOT NULL,
    expires_at  INTEGER DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_tokens_token ON tokens(token);

CREATE TABLE IF NOT EXISTS albums (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    description TEXT DEFAULT '',
    is_public   INTEGER NOT NULL DEFAULT 0,
    cover_key   TEXT DEFAULT '',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_albums_user ON albums(user_id);

CREATE TABLE IF NOT EXISTS images (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL DEFAULT 0,    -- 0 = 匿名
    album_id    INTEGER NOT NULL DEFAULT 0,
    key         TEXT NOT NULL UNIQUE,           -- 存储 key 如 2026/04/15/a3f2e1.webp
    filename    TEXT NOT NULL,                  -- 原始文件名
    mime        TEXT NOT NULL,
    size        INTEGER NOT NULL,
    width       INTEGER NOT NULL DEFAULT 0,
    height      INTEGER NOT NULL DEFAULT 0,
    hash        TEXT NOT NULL DEFAULT '',       -- sha256
    thumb_small TEXT DEFAULT '',
    thumb_med   TEXT DEFAULT '',
    tags        TEXT DEFAULT '',                -- 逗号分隔
    ip          TEXT DEFAULT '',
    views       INTEGER NOT NULL DEFAULT 0,
    created_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_images_user ON images(user_id);
CREATE INDEX IF NOT EXISTS idx_images_album ON images(album_id);
CREATE INDEX IF NOT EXISTS idx_images_created ON images(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_images_hash ON images(hash);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL DEFAULT 0,
    action     TEXT NOT NULL,
    target     TEXT DEFAULT '',
    ip         TEXT DEFAULT '',
    detail     TEXT DEFAULT '',
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_logs(created_at DESC);
