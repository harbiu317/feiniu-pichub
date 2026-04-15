package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/levis/pichub/internal/config"
	server "github.com/levis/pichub/internal/server"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	var (
		configPath = flag.String("config", "config.yaml", "配置文件路径")
		showVer    = flag.Bool("version", false, "显示版本")
	)
	flag.Parse()

	if *showVer {
		log.Printf("PicHub %s (%s)\n", version, commit)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	server.Version = version

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("初始化服务失败: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("PicHub %s 启动于 http://%s", version, cfg.Server.Addr)
		if err := srv.Start(); err != nil {
			log.Fatalf("服务运行失败: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("收到关闭信号，正在停止…")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("关闭异常: %v", err)
		os.Exit(1)
	}
}
