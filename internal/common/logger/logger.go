package logger

import (
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"tg_cloud_server/internal/common/config"
)

var (
	globalLogger *LoggerManager
	once         sync.Once
)

// LoggerManager 日志管理器
type LoggerManager struct {
	mainLogger  *zap.Logger
	errorLogger *zap.Logger
	warnLogger  *zap.Logger
	infoLogger  *zap.Logger
	debugLogger *zap.Logger
	taskLogger  *zap.Logger
	apiLogger   *zap.Logger
	config      *config.LoggingConfig
}

// Init 初始化日志管理器
func Init(config *config.LoggingConfig) error {
	var err error
	once.Do(func() {
		globalLogger, err = newLoggerManager(config)
	})
	return err
}

// Get 获取主日志器
func Get() *zap.Logger {
	if globalLogger == nil {
		// 如果logger未初始化，使用默认配置
		defaultConfig := &config.LoggingConfig{
			Level:      "info",
			Format:     "json", 
			Output:     "file",
			Filename:   "logs/app.log",
			MaxSize:    100,
			MaxBackups: 7,
			MaxAge:     30,
			Compress:   true,
		}
		Init(defaultConfig)
	}
	return globalLogger.mainLogger
}

// GetError 获取错误日志器
func GetError() *zap.Logger {
	if globalLogger == nil {
		Get() // 初始化
	}
	return globalLogger.errorLogger
}

// GetWarn 获取警告日志器
func GetWarn() *zap.Logger {
	if globalLogger == nil {
		Get() // 初始化
	}
	return globalLogger.warnLogger
}

// GetInfo 获取信息日志器
func GetInfo() *zap.Logger {
	if globalLogger == nil {
		Get() // 初始化
	}
	return globalLogger.infoLogger
}

// GetDebug 获取调试日志器
func GetDebug() *zap.Logger {
	if globalLogger == nil {
		Get() // 初始化
	}
	return globalLogger.debugLogger
}

// GetTask 获取任务专用日志器
func GetTask() *zap.Logger {
	if globalLogger == nil {
		Get() // 初始化
	}
	return globalLogger.taskLogger
}

// GetAPI 获取API专用日志器
func GetAPI() *zap.Logger {
	if globalLogger == nil {
		Get() // 初始化
	}
	return globalLogger.apiLogger
}

// newLoggerManager 创建日志管理器
func newLoggerManager(config *config.LoggingConfig) (*LoggerManager, error) {
	// 确保日志目录存在
	if err := ensureLogDir(config); err != nil {
		return nil, err
	}

	manager := &LoggerManager{
		config: config,
	}

	// 创建各级别日志器
	var err error
	manager.mainLogger, err = createLogger(config, config.Filename, zapcore.InfoLevel)
	if err != nil {
		return nil, err
	}

	manager.errorLogger, err = createLogger(config, config.Files.ErrorLog, zapcore.ErrorLevel)
	if err != nil {
		return nil, err
	}

	manager.warnLogger, err = createLogger(config, config.Files.WarnLog, zapcore.WarnLevel)
	if err != nil {
		return nil, err
	}

	manager.infoLogger, err = createLogger(config, config.Files.InfoLog, zapcore.InfoLevel)
	if err != nil {
		return nil, err
	}

	manager.debugLogger, err = createLogger(config, config.Files.DebugLog, zapcore.DebugLevel)
	if err != nil {
		return nil, err
	}

	// 任务和API日志器使用Info级别，但写入独立文件
	manager.taskLogger, err = createLogger(config, config.Files.TaskLog, zapcore.InfoLevel)
	if err != nil {
		return nil, err
	}

	manager.apiLogger, err = createLogger(config, config.Files.APILog, zapcore.InfoLevel)
	if err != nil {
		return nil, err
	}

	return manager, nil
}

// createLogger 创建单个日志器
func createLogger(config *config.LoggingConfig, filename string, level zapcore.Level) (*zap.Logger, error) {
	// 设置编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 选择编码器
	var encoder zapcore.Encoder
	if config.Format == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 设置写入器
	var writer zapcore.WriteSyncer
	if config.Output == "stdout" {
		writer = zapcore.AddSync(os.Stdout)
	} else {
		// 使用 lumberjack 进行日志轮转
		lumberjackLogger := &lumberjack.Logger{
			Filename:   filename,
			MaxSize:    config.MaxSize,    // MB
			MaxBackups: config.MaxBackups, // 保留的旧文件数量
			MaxAge:     config.MaxAge,     // 天数
			Compress:   config.Compress,   // 压缩旧文件
			LocalTime:  true,              // 使用本地时间
		}
		writer = zapcore.AddSync(lumberjackLogger)
	}

	// 创建核心
	core := zapcore.NewCore(encoder, writer, level)

	// 创建logger，添加调用者信息和错误堆栈
	logger := zap.New(core, 
		zap.AddCaller(), 
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCallerSkip(1), // 跳过一级调用栈
	)

	return logger, nil
}

// ensureLogDir 确保日志目录存在
func ensureLogDir(config *config.LoggingConfig) error {
	logFiles := []string{
		config.Filename,
		config.Files.ErrorLog,
		config.Files.WarnLog, 
		config.Files.InfoLog,
		config.Files.DebugLog,
		config.Files.TaskLog,
		config.Files.APILog,
	}

	for _, filename := range logFiles {
		if filename == "" {
			continue
		}
		
		dir := filepath.Dir(filename)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// Sync 同步所有日志缓冲区
func Sync() {
	if globalLogger != nil {
		globalLogger.mainLogger.Sync()
		globalLogger.errorLogger.Sync()
		globalLogger.warnLogger.Sync()
		globalLogger.infoLogger.Sync()
		globalLogger.debugLogger.Sync()
		globalLogger.taskLogger.Sync()
		globalLogger.apiLogger.Sync()
	}
}

// LogTask 记录任务相关日志
func LogTask(level zapcore.Level, msg string, fields ...zap.Field) {
	if globalLogger == nil {
		Get() // 初始化
	}
	
	// 同时写入主日志和任务日志
	switch level {
	case zapcore.DebugLevel:
		globalLogger.taskLogger.Debug(msg, fields...)
		globalLogger.mainLogger.Debug(msg, fields...)
	case zapcore.InfoLevel:
		globalLogger.taskLogger.Info(msg, fields...)
		globalLogger.mainLogger.Info(msg, fields...)
	case zapcore.WarnLevel:
		globalLogger.taskLogger.Warn(msg, fields...)
		globalLogger.mainLogger.Warn(msg, fields...)
		globalLogger.warnLogger.Warn(msg, fields...)
	case zapcore.ErrorLevel:
		globalLogger.taskLogger.Error(msg, fields...)
		globalLogger.mainLogger.Error(msg, fields...)
		globalLogger.errorLogger.Error(msg, fields...)
	}
}

// LogAPI 记录API相关日志
func LogAPI(level zapcore.Level, msg string, fields ...zap.Field) {
	if globalLogger == nil {
		Get() // 初始化
	}
	
	// 同时写入主日志和API日志
	switch level {
	case zapcore.DebugLevel:
		globalLogger.apiLogger.Debug(msg, fields...)
		globalLogger.mainLogger.Debug(msg, fields...)
	case zapcore.InfoLevel:
		globalLogger.apiLogger.Info(msg, fields...)
		globalLogger.mainLogger.Info(msg, fields...)
	case zapcore.WarnLevel:
		globalLogger.apiLogger.Warn(msg, fields...)
		globalLogger.mainLogger.Warn(msg, fields...)
		globalLogger.warnLogger.Warn(msg, fields...)
	case zapcore.ErrorLevel:
		globalLogger.apiLogger.Error(msg, fields...)
		globalLogger.mainLogger.Error(msg, fields...)
		globalLogger.errorLogger.Error(msg, fields...)
	}
}
