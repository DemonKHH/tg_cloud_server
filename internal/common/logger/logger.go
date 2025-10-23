package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"tg_cloud_server/internal/common/config"
)

var (
	logger *zap.Logger
	once   sync.Once
)

// Init 初始化日志器
func Init(config *config.LoggingConfig) error {
	var err error
	once.Do(func() {
		logger, err = createLogger(config)
	})
	return err
}

// Get 获取全局日志器
func Get() *zap.Logger {
	if logger == nil {
		// 如果logger未初始化，使用默认配置
		defaultConfig := &config.LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		}
		Init(defaultConfig)
	}
	return logger
}

// createLogger 创建日志器
func createLogger(config *config.LoggingConfig) (*zap.Logger, error) {
	// 设置日志级别
	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

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
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 选择编码器
	var encoder zapcore.Encoder
	if config.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 设置输出
	var writer zapcore.WriteSyncer
	if config.Output == "stdout" {
		writer = zapcore.AddSync(os.Stdout)
	} else if config.Filename != "" {
		// 这里可以添加文件轮转逻辑
		file, err := os.OpenFile(config.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		writer = zapcore.AddSync(file)
	} else {
		writer = zapcore.AddSync(os.Stdout)
	}

	// 创建核心
	core := zapcore.NewCore(encoder, writer, level)

	// 创建logger
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger, nil
}

// Sync 同步日志缓冲区
func Sync() {
	if logger != nil {
		logger.Sync()
	}
}
