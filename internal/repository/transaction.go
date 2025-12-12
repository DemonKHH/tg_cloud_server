package repository

import (
	"context"

	"gorm.io/gorm"
)

// TransactionManager 事务管理器接口
type TransactionManager interface {
	// WithTransaction 在事务中执行函数
	WithTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	// GetDB 获取数据库实例
	GetDB() *gorm.DB
}

// transactionManager 事务管理器实现
type transactionManager struct {
	db *gorm.DB
}

// NewTransactionManager 创建事务管理器
func NewTransactionManager(db *gorm.DB) TransactionManager {
	return &transactionManager{db: db}
}

// WithTransaction 在事务中执行函数
func (tm *transactionManager) WithTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// GetDB 获取数据库实例
func (tm *transactionManager) GetDB() *gorm.DB {
	return tm.db
}
