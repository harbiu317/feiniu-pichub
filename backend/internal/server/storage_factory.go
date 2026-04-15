package server

import (
	"github.com/levis/pichub/internal/config"
	"github.com/levis/pichub/internal/storage"
)

// buildStorage 根据配置的 driver 名构造存储后端。
// 七牛/阿里云/腾讯云 均走 S3 兼容接口，共用 s3 驱动。
func buildStorage(cfg *config.Config) (storage.Storage, error) {
	switch cfg.Storage.Driver {
	case "s3", "qiniu", "aliyun", "tencent":
		return storage.NewS3(cfg.Storage.S3)
	default:
		return storage.NewLocal(cfg.Storage.Local.Root, cfg.Server.PublicURL), nil
	}
}
