package logger

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"tg_cloud_server/internal/common/config"
)

// LogManager 日志管理器
type LogManager struct {
	config *config.LoggingConfig
}

// NewLogManager 创建日志管理器
func NewLogManager(config *config.LoggingConfig) *LogManager {
	return &LogManager{
		config: config,
	}
}

// CleanupOldLogs 清理过期日志文件
func (lm *LogManager) CleanupOldLogs() error {
	logFiles := []string{
		lm.config.Filename,
		lm.config.Files.ErrorLog,
		lm.config.Files.WarnLog,
		lm.config.Files.InfoLog,
		lm.config.Files.DebugLog,
		lm.config.Files.TaskLog,
		lm.config.Files.APILog,
	}

	cutoffTime := time.Now().AddDate(0, 0, -lm.config.MaxAge)
	totalCleaned := 0

	for _, logFile := range logFiles {
		if logFile == "" {
			continue
		}

		cleaned, err := lm.cleanupLogFile(logFile, cutoffTime)
		if err != nil {
			Get().Error("Failed to cleanup log file",
				zap.String("file", logFile),
				zap.Error(err))
			continue
		}
		totalCleaned += cleaned
	}

	if totalCleaned > 0 {
		Get().Info("Log cleanup completed",
			zap.Int("files_cleaned", totalCleaned),
			zap.Time("cutoff_time", cutoffTime))
	}

	return nil
}

// cleanupLogFile 清理单个日志文件的历史版本
func (lm *LogManager) cleanupLogFile(logFile string, cutoffTime time.Time) (int, error) {
	logDir := filepath.Dir(logFile)
	baseName := filepath.Base(logFile)
	
	// 扫描日志目录
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // 目录不存在，跳过
		}
		return 0, err
	}

	cleaned := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		// 检查是否是相关的日志文件（包括轮转文件）
		if lm.isRelatedLogFile(fileName, baseName) {
			filePath := filepath.Join(logDir, fileName)
			info, err := entry.Info()
			if err != nil {
				continue
			}

			// 如果文件过期，删除它
			if info.ModTime().Before(cutoffTime) {
				if err := os.Remove(filePath); err != nil {
					Get().Warn("Failed to remove old log file",
						zap.String("file", filePath),
						zap.Error(err))
				} else {
					Get().Debug("Removed old log file",
						zap.String("file", filePath),
						zap.Time("mod_time", info.ModTime()))
					cleaned++
				}
			}
		}
	}

	return cleaned, nil
}

// isRelatedLogFile 检查文件是否为相关日志文件
func (lm *LogManager) isRelatedLogFile(fileName, baseName string) bool {
	// 去掉扩展名
	baseNameNoExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	
	// 检查文件名模式
	patterns := []string{
		baseName,                                    // app.log
		baseNameNoExt + "-*",                        // app-2024-01-01.log
		baseNameNoExt + ".*",                        // app.1.log
		baseNameNoExt + "-*.gz",                     // app-2024-01-01.log.gz
		baseNameNoExt + ".*.gz",                     // app.1.log.gz
	}

	for _, pattern := range patterns {
		matched, _ := filepath.Match(pattern, fileName)
		if matched {
			return true
		}
	}

	// 额外检查lumberjack的命名模式
	if strings.HasPrefix(fileName, baseNameNoExt) {
		return true
	}

	return false
}

// GetLogFiles 获取所有日志文件列表
func (lm *LogManager) GetLogFiles() []LogFileInfo {
	logFiles := []string{
		lm.config.Filename,
		lm.config.Files.ErrorLog,
		lm.config.Files.WarnLog,
		lm.config.Files.InfoLog,
		lm.config.Files.DebugLog,
		lm.config.Files.TaskLog,
		lm.config.Files.APILog,
	}

	var results []LogFileInfo

	for _, logFile := range logFiles {
		if logFile == "" {
			continue
		}

		info := lm.getLogFileInfo(logFile)
		if info != nil {
			results = append(results, *info)
		}
	}

	return results
}

// LogFileInfo 日志文件信息
type LogFileInfo struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	IsCompressed bool      `json:"is_compressed"`
	Type         string    `json:"type"`
}

// getLogFileInfo 获取日志文件信息
func (lm *LogManager) getLogFileInfo(logFile string) *LogFileInfo {
	info, err := os.Stat(logFile)
	if err != nil {
		return nil
	}

	logType := lm.getLogTypeFromPath(logFile)
	isCompressed := strings.HasSuffix(logFile, ".gz")

	return &LogFileInfo{
		Path:         logFile,
		Size:         info.Size(),
		ModTime:      info.ModTime(),
		IsCompressed: isCompressed,
		Type:         logType,
	}
}

// getLogTypeFromPath 从路径获取日志类型
func (lm *LogManager) getLogTypeFromPath(path string) string {
	switch {
	case strings.Contains(path, "error"):
		return "error"
	case strings.Contains(path, "warn"):
		return "warn"
	case strings.Contains(path, "info"):
		return "info"
	case strings.Contains(path, "debug"):
		return "debug"
	case strings.Contains(path, "task"):
		return "task"
	case strings.Contains(path, "api"):
		return "api"
	default:
		return "main"
	}
}

// RotateLogs 手动轮转日志
func (lm *LogManager) RotateLogs() error {
	// 这里可以实现手动轮转逻辑
	// lumberjack会自动处理轮转，但可以提供手动接口
	
	Get().Info("Manual log rotation requested")
	
	// 同步所有日志缓冲区
	Sync()
	
	Get().Info("Log rotation completed")
	return nil
}

// GetLogStats 获取日志统计信息
func (lm *LogManager) GetLogStats() LogStats {
	files := lm.GetLogFiles()
	
	stats := LogStats{
		TotalFiles: len(files),
		TotalSize:  0,
		FilesByType: make(map[string]int),
	}

	for _, file := range files {
		stats.TotalSize += file.Size
		stats.FilesByType[file.Type]++
	}

	return stats
}

// LogStats 日志统计信息
type LogStats struct {
	TotalFiles  int            `json:"total_files"`
	TotalSize   int64          `json:"total_size"`
	FilesByType map[string]int `json:"files_by_type"`
}

// LogTaskWithFields 带预设字段的任务日志记录
func LogTaskWithFields(level zapcore.Level, msg string, taskID uint64, taskType string, additionalFields ...zap.Field) {
	fields := []zap.Field{
		zap.Uint64("task_id", taskID),
		zap.String("task_type", taskType),
		zap.Time("timestamp", time.Now()),
	}
	
	fields = append(fields, additionalFields...)
	LogTask(level, msg, fields...)
}

// LogAPIWithFields 带预设字段的API日志记录  
func LogAPIWithFields(level zapcore.Level, msg string, method, path string, statusCode int, additionalFields ...zap.Field) {
	fields := []zap.Field{
		zap.String("method", method),
		zap.String("path", path),
		zap.Int("status_code", statusCode),
		zap.Time("timestamp", time.Now()),
	}
	
	fields = append(fields, additionalFields...)
	LogAPI(level, msg, fields...)
}
