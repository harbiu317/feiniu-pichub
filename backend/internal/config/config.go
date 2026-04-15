package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Storage  StorageConfig  `yaml:"storage"`
	Auth     AuthConfig     `yaml:"auth"`
	Image    ImageConfig    `yaml:"image"`
	Limits   LimitsConfig   `yaml:"limits"`
}

type ServerConfig struct {
	Addr        string `yaml:"addr"`
	PublicURL   string `yaml:"public_url"`
	TrustProxy  bool   `yaml:"trust_proxy"`
	AllowAnon   bool   `yaml:"allow_anonymous"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type StorageConfig struct {
	Driver string `yaml:"driver"` // local | s3 | qiniu | aliyun | tencent
	Local  struct {
		Root string `yaml:"root"`
	} `yaml:"local"`
	S3 S3Config `yaml:"s3"`
}

// S3Config 通用 S3 兼容配置，覆盖七牛/阿里/腾讯/AWS/MinIO/R2 等。
type S3Config struct {
	Endpoint   string `yaml:"endpoint"`    // 如 s3.cn-hangzhou.aliyuncs.com（不带协议）
	Region     string `yaml:"region"`      // 如 cn-hangzhou / us-east-1
	Bucket     string `yaml:"bucket"`
	AccessKey  string `yaml:"access_key"`
	SecretKey  string `yaml:"secret_key"`
	UseSSL     bool   `yaml:"use_ssl"`
	PathStyle  bool   `yaml:"path_style"`  // MinIO / 某些自建需要 true
	PublicBase string `yaml:"public_base"` // 对外访问前缀，如 https://cdn.example.com 或 https://bucket.oss-cn-hangzhou.aliyuncs.com
	Prefix     string `yaml:"prefix"`      // 对象 key 前缀
}

type AuthConfig struct {
	JWTSecret     string `yaml:"jwt_secret"`
	TokenLifetime int    `yaml:"token_lifetime_hours"`
	AdminUser     string `yaml:"admin_user"`
	AdminPassword string `yaml:"admin_password"`
}

type ImageConfig struct {
	MaxSizeMB     int      `yaml:"max_size_mb"`
	AllowedTypes  []string `yaml:"allowed_types"`
	AutoCompress  bool     `yaml:"auto_compress"`
	Quality       int      `yaml:"quality"`
	ConvertWebP   bool     `yaml:"convert_webp"`
	StripEXIF     bool     `yaml:"strip_exif"`
	Watermark     struct {
		Enabled bool   `yaml:"enabled"`
		Text    string `yaml:"text"`
		Opacity int    `yaml:"opacity"`
		Font    string `yaml:"font"` // 可选 TTF/TTC 字体文件路径。留空则自动探测系统 CJK 字体
	} `yaml:"watermark"`
	Thumbnail struct {
		Small  int `yaml:"small"`
		Medium int `yaml:"medium"`
	} `yaml:"thumbnail"`
}

type LimitsConfig struct {
	UploadPerMinute int      `yaml:"upload_per_minute"`
	IPWhitelist     []string `yaml:"ip_whitelist"`
	IPBlacklist     []string `yaml:"ip_blacklist"`
}

// Load 读取配置文件；若不存在则生成默认配置并写入指定路径。
func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := Default()
		if err := Save(path, cfg); err != nil {
			return nil, fmt.Errorf("写入默认配置失败: %w", err)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 环境变量覆盖（fpk 场景有用）
	if v := os.Getenv("PICHUB_ADDR"); v != "" {
		cfg.Server.Addr = v
	}
	if v := os.Getenv("PICHUB_ADMIN_USER"); v != "" {
		cfg.Auth.AdminUser = v
	}
	if v := os.Getenv("PICHUB_ADMIN_PASSWORD"); v != "" {
		cfg.Auth.AdminPassword = v
	}
	if v := os.Getenv("PICHUB_DATA_DIR"); v != "" {
		cfg.Database.Path = filepath.Join(v, "pichub.db")
		cfg.Storage.Local.Root = filepath.Join(v, "uploads")
	}
	if v := os.Getenv("PICHUB_PUBLIC_URL"); v != "" {
		cfg.Server.PublicURL = v
	}
	if v := os.Getenv("PICHUB_ALLOW_ANON"); v != "" {
		cfg.Server.AllowAnon = v == "true" || v == "1"
	}
	if v := os.Getenv("PICHUB_MAX_SIZE_MB"); v != "" {
		var n int
		_, _ = fmt.Sscanf(v, "%d", &n)
		if n > 0 {
			cfg.Image.MaxSizeMB = n
		}
	}

	return cfg, nil
}

func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func Default() *Config {
	c := &Config{}
	c.Server.Addr = "0.0.0.0:7800"
	c.Server.PublicURL = "http://localhost:7800"
	c.Server.AllowAnon = false
	c.Database.Path = "data/pichub.db"
	c.Storage.Driver = "local"
	c.Storage.Local.Root = "data/uploads"
	c.Auth.JWTSecret = randomSecret(48)
	c.Auth.TokenLifetime = 24 * 30
	c.Auth.AdminUser = ""     // 首次启动向导设置
	c.Auth.AdminPassword = ""
	c.Image.MaxSizeMB = 20
	c.Image.AllowedTypes = []string{"image/jpeg", "image/png", "image/webp", "image/gif"}
	c.Image.AutoCompress = true
	c.Image.Quality = 85
	c.Image.ConvertWebP = false
	c.Image.StripEXIF = true
	c.Image.Thumbnail.Small = 200
	c.Image.Thumbnail.Medium = 600
	c.Limits.UploadPerMinute = 30
	return c
}
