package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"friends-records/config"
	"friends-records/internal/apperror"
	modelmysql "friends-records/internal/models/mysql"
	"friends-records/internal/token"
	"friends-records/router"
	"friends-records/webassets"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	if err := run(); err != nil {
		log.Fatalf("程序启动失败：%v", err)
	}
}
func run() error {
	cfg, err := config.Load("config.toml")
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := modelmysql.Open(ctx, cfg.MySQL.DSN)
	if err != nil {
		return err
	}
	if db != nil {
		defer db.Close()
	}
	templates, err := webassets.Templates()
	if err != nil {
		return apperror.Wrap(err, "加载后台HTML模板失败")
	}
	static, err := webassets.Static()
	if err != nil {
		return apperror.Wrap(err, "加载后台静态资源失败")
	}
	tokens := token.New(cfg.App.TokenSecret)
	if cfg.App.TokenSecret == "" {
		tokens, err = token.NewRandom()
		if err != nil {
			return err
		}
		log.Printf("提示：开发环境未填写app.token_secret，已生成临时密钥；程序重启后旧Token会失效")
	}
	server := &http.Server{Addr: cfg.App.Addr, Handler: router.New(cfg, db, templates, static, tokens), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("服务开始监听：%s", cfg.App.Addr)
		serverErrors <- apperror.Wrap(server.ListenAndServe(), fmt.Sprintf("HTTP服务监听地址%s失败", cfg.App.Addr))
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return apperror.Wrap(err, "HTTP服务停止失败")
		}
		return nil
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
