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

	var cfg = core.Conf.Server
	var srv = &http.Server{
		Addr:         cfg.Addr,
		Handler:      router.New(cfg),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	var errCh = make(chan error, 1)
	var quit = make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		logger.Printf("HTTP 服务器启动 [%s]\n", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		logger.Printf("服务器错误: %v\n", err)
		os.Exit(1)
	case sig := <-quit:
		logger.Printf("收到关闭信号: %v\n", sig)
	}

	var shutdownCtx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Printf("服务器关闭失败: %v\n", err)
	}
	logger.Printf("服务器优雅关闭\n")
}
