package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"tg_cloud_server/internal/common/logger"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LogLevel 日志级别
type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelDebug LogLevel = "debug"
)

// ValidLogLevels 有效的日志级别列表
var ValidLogLevels = map[LogLevel]bool{
	LogLevelInfo:  true,
	LogLevelWarn:  true,
	LogLevelError: true,
	LogLevelDebug: true,
}

// IsValidLogLevel 检查日志级别是否有效
func IsValidLogLevel(level LogLevel) bool {
	return ValidLogLevels[level]
}

// TaskLogEntry 任务日志条目
type TaskLogEntry struct {
	ID        uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	TaskID    uint64          `json:"task_id" gorm:"not null;index:idx_task_created"`
	AccountID *uint64         `json:"account_id" gorm:"index"`
	Level     LogLevel        `json:"level" gorm:"size:10;default:'info'"`
	Action    string          `json:"action" gorm:"size:50;not null"`
	Message   string          `json:"message" gorm:"type:text"`
	ExtraData json.RawMessage `json:"extra_data" gorm:"type:json"`
	CreatedAt time.Time       `json:"created_at" gorm:"index:idx_task_created"`
}

// TableName 指定表名
func (TaskLogEntry) TableName() string {
	return "task_logs"
}

// Validate 验证日志条目
func (e *TaskLogEntry) Validate() error {
	if e.TaskID == 0 {
		return errors.New("task_id is required")
	}
	if e.Action == "" {
		return errors.New("action is required")
	}
	if !IsValidLogLevel(e.Level) {
		return fmt.Errorf("invalid log level: %s", e.Level)
	}
	return nil
}

// LogQueryFilter 日志查询过滤器
type LogQueryFilter struct {
	TaskID    uint64     `json:"task_id"`
	AccountID *uint64    `json:"account_id,omitempty"`
	Level     *LogLevel  `json:"level,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Page      int        `json:"page"`
	Limit     int        `json:"limit"`
	Order     string     `json:"order"` // "asc" or "desc"
}

// Normalize 规范化过滤器参数
func (f *LogQueryFilter) Normalize() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 50
	}
	if f.Limit > 200 {
		f.Limit = 200
	}
	if f.Order != "desc" {
		f.Order = "asc"
	}
}

// LogQueryResult 日志查询结果
type LogQueryResult struct {
	Logs    []*TaskLogEntry `json:"logs"`
	Total   int64           `json:"total"`
	Page    int             `json:"page"`
	Limit   int             `json:"limit"`
	HasMore bool            `json:"has_more"`
}

// TaskLogService 任务日志服务接口
type TaskLogService interface {
	// CreateLog 创建日志并推送
	CreateLog(ctx context.Context, log *TaskLogEntry) error

	// BatchCreateLogs 批量创建日志
	BatchCreateLogs(ctx context.Context, logs []*TaskLogEntry) error

	// QueryLogs 查询日志（支持分页和过滤）
	QueryLogs(ctx context.Context, filter *LogQueryFilter) (*LogQueryResult, error)

	// GetRecentLogs 获取任务最近的日志
	GetRecentLogs(ctx context.Context, taskID uint64, limit int) ([]*TaskLogEntry, error)

	// CleanupExpiredLogs 清理过期日志
	CleanupExpiredLogs(ctx context.Context, retentionDays int) (int64, error)

	// DeleteTaskLogs 删除任务相关日志
	DeleteTaskLogs(ctx context.Context, taskID uint64) error
}

// LogPusher 日志推送接口（用于解耦NotificationService）
type LogPusher interface {
	// PushTaskLog 推送任务日志给订阅者
	PushTaskLog(taskID uint64, log *TaskLogEntry)
}

// taskLogService 任务日志服务实现
type taskLogService struct {
	db        *gorm.DB
	logPusher LogPusher
	logger    *zap.Logger
	mutex     sync.RWMutex
}

// NewTaskLogService 创建任务日志服务
func NewTaskLogService(db *gorm.DB, logPusher LogPusher) TaskLogService {
	return &taskLogService{
		db:        db,
		logPusher: logPusher,
		logger:    logger.Get().Named("task_log_service"),
	}
}

// CreateLog 创建日志并推送
func (s *taskLogService) CreateLog(ctx context.Context, log *TaskLogEntry) error {
	s.logger.Info("CreateLog called",
		zap.Uint64("task_id", log.TaskID),
		zap.String("action", log.Action),
		zap.String("message", log.Message))

	// 设置默认值
	if log.Level == "" {
		log.Level = LogLevelInfo
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}

	// 验证日志条目
	if err := log.Validate(); err != nil {
		s.logger.Warn("Invalid log entry", zap.Error(err))
		return fmt.Errorf("invalid log entry: %w", err)
	}

	// 先持久化到数据库
	if err := s.db.WithContext(ctx).Create(log).Error; err != nil {
		s.logger.Error("Failed to create task log",
			zap.Uint64("task_id", log.TaskID),
			zap.String("action", log.Action),
			zap.Error(err))
		return fmt.Errorf("failed to create task log: %w", err)
	}

	s.logger.Info("Task log created successfully",
		zap.Uint64("log_id", log.ID),
		zap.Uint64("task_id", log.TaskID),
		zap.String("level", string(log.Level)),
		zap.String("action", log.Action))

	// 推送给订阅者（异步，不阻塞）
	if s.logPusher != nil {
		go s.logPusher.PushTaskLog(log.TaskID, log)
	} else {
		s.logger.Warn("No log pusher configured, skipping push")
	}

	return nil
}

// BatchCreateLogs 批量创建日志
func (s *taskLogService) BatchCreateLogs(ctx context.Context, logs []*TaskLogEntry) error {
	if len(logs) == 0 {
		return nil
	}

	// 验证并设置默认值
	now := time.Now()
	for _, log := range logs {
		if log.Level == "" {
			log.Level = LogLevelInfo
		}
		if log.CreatedAt.IsZero() {
			log.CreatedAt = now
		}
		if err := log.Validate(); err != nil {
			return fmt.Errorf("invalid log entry: %w", err)
		}
	}

	// 批量插入
	if err := s.db.WithContext(ctx).CreateInBatches(logs, 100).Error; err != nil {
		s.logger.Error("Failed to batch create task logs",
			zap.Int("count", len(logs)),
			zap.Error(err))
		return fmt.Errorf("failed to batch create task logs: %w", err)
	}

	s.logger.Debug("Task logs batch created", zap.Int("count", len(logs)))

	// 推送给订阅者
	if s.logPusher != nil {
		for _, log := range logs {
			go s.logPusher.PushTaskLog(log.TaskID, log)
		}
	}

	return nil
}

// QueryLogs 查询日志（支持分页和过滤）
func (s *taskLogService) QueryLogs(ctx context.Context, filter *LogQueryFilter) (*LogQueryResult, error) {
	// 规范化过滤器参数
	filter.Normalize()

	query := s.db.WithContext(ctx).Model(&TaskLogEntry{}).Where("task_id = ?", filter.TaskID)

	// 应用过滤条件
	if filter.AccountID != nil {
		query = query.Where("account_id = ?", *filter.AccountID)
	}
	if filter.Level != nil {
		query = query.Where("level = ?", *filter.Level)
	}
	if filter.StartTime != nil {
		query = query.Where("created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("created_at <= ?", *filter.EndTime)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		s.logger.Error("Failed to count task logs",
			zap.Uint64("task_id", filter.TaskID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to count task logs: %w", err)
	}

	// 排序
	orderClause := "created_at ASC"
	if filter.Order == "desc" {
		orderClause = "created_at DESC"
	}

	// 分页查询
	offset := (filter.Page - 1) * filter.Limit
	var logs []*TaskLogEntry
	if err := query.Order(orderClause).Offset(offset).Limit(filter.Limit).Find(&logs).Error; err != nil {
		s.logger.Error("Failed to query task logs",
			zap.Uint64("task_id", filter.TaskID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to query task logs: %w", err)
	}

	hasMore := int64(offset+len(logs)) < total

	return &LogQueryResult{
		Logs:    logs,
		Total:   total,
		Page:    filter.Page,
		Limit:   filter.Limit,
		HasMore: hasMore,
	}, nil
}

// GetRecentLogs 获取任务最近的日志
func (s *taskLogService) GetRecentLogs(ctx context.Context, taskID uint64, limit int) ([]*TaskLogEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var logs []*TaskLogEntry
	if err := s.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		s.logger.Error("Failed to get recent task logs",
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get recent task logs: %w", err)
	}

	// 反转顺序，使最旧的在前面
	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}

	return logs, nil
}

// CleanupExpiredLogs 清理过期日志
func (s *taskLogService) CleanupExpiredLogs(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 30 // 默认保留30天
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	result := s.db.WithContext(ctx).
		Where("created_at < ?", cutoffTime).
		Delete(&TaskLogEntry{})

	if result.Error != nil {
		s.logger.Error("Failed to cleanup expired task logs",
			zap.Int("retention_days", retentionDays),
			zap.Time("cutoff_time", cutoffTime),
			zap.Error(result.Error))
		return 0, fmt.Errorf("failed to cleanup expired task logs: %w", result.Error)
	}

	s.logger.Info("Expired task logs cleaned up",
		zap.Int64("deleted_count", result.RowsAffected),
		zap.Int("retention_days", retentionDays),
		zap.Time("cutoff_time", cutoffTime))

	return result.RowsAffected, nil
}

// DeleteTaskLogs 删除任务相关日志
func (s *taskLogService) DeleteTaskLogs(ctx context.Context, taskID uint64) error {
	result := s.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Delete(&TaskLogEntry{})

	if result.Error != nil {
		s.logger.Error("Failed to delete task logs",
			zap.Uint64("task_id", taskID),
			zap.Error(result.Error))
		return fmt.Errorf("failed to delete task logs: %w", result.Error)
	}

	s.logger.Info("Task logs deleted",
		zap.Uint64("task_id", taskID),
		zap.Int64("deleted_count", result.RowsAffected))

	return nil
}
