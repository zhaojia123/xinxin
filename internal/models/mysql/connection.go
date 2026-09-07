package mysql

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"friends-records/internal/apperror"
	_ "github.com/go-sql-driver/mysql"
)

func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, nil
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, apperror.Wrap(err, "打开MySQL连接失败")
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, apperror.Wrap(err, "连接MySQL失败，请检查地址、端口、数据库名、用户名和密码")
	}
	return db, nil
}
