package pkg

import (
	"database/sql"
	"errors"
	"time"
)

// InitMySQL 初始化GORM的MySQL连接
func InitMySQL(dsn string) (*sql.DB, error) {
	// 打开数据库连接（不立即校验，需Ping）
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.New("连接MySQL失败：" + err.Error())
	}

	// 校验连接是否可用
	if err := db.Ping(); err != nil {
		db.Close() // 关闭无效连接
		return nil, errors.New("校验MySQL连接失败：" + err.Error())
	}

	// 设置连接池参数（优化性能）
	db.SetMaxOpenConns(20)                  // 最大打开连接数
	db.SetMaxIdleConns(10)                  // 最大空闲连接数
	db.SetConnMaxLifetime(1)                // 连接最大存活时间
	db.SetConnMaxIdleTime(30 * time.Minute) // 连接最大空闲时间

	return db, nil
}
