package main

import (
	"context"
	"errors"
	"gin-demo/src/api/router"
	"gin-demo/src/core"
	"gin-demo/src/pkg/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var configFile string

func main() {

	defer core.Cleanup()

	ctx := context.Background()
	cfg := core.Conf.Server
	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      router.New(cfg),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Infof(ctx, "HTTP 服务器启动 [%s]", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-errCh:
		logger.Errorf(ctx, "服务器错误: %v", err)
		os.Exit(1)
	case sig := <-quit:
		logger.Infof(ctx, "收到关闭信号: %v", sig)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Errorf(ctx, "服务器关闭失败: %v", err)
	}
	logger.Infof(ctx, "服务器优雅关闭")
}
