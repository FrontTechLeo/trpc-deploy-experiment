package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// InitMySQL 初始化 MySQL 连接池
func InitMySQL(dsn string) (*sql.DB, error) {
	if dsn == "" {
		dsn = "root:root@tcp(127.0.0.1:3306)/gametool?parseTime=true&charset=utf8mb4"
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql failed: %w", err)
	}

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(10 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql failed: %w", err)
	}
	return db, nil
}
