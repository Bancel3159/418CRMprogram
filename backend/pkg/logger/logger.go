package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var L *zap.Logger

// Init 初始化全局 logger
func Init(level string) {
	var l zapcore.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		l = zapcore.InfoLevel
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(l)
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	L, _ = cfg.Build()
}

// ========== 便捷方法(供各模块直接使用) ==========

func Info(msg string, fields ...zap.Field) {
	if L != nil {
		L.Info(msg, fields...)
	}
}

func Warn(msg string, fields ...zap.Field) {
	if L != nil {
		L.Warn(msg, fields...)
	}
}

func Error(msg string, fields ...zap.Field) {
	if L != nil {
		L.Error(msg, fields...)
	}
}

func Debug(msg string, fields ...zap.Field) {
	if L != nil {
		L.Debug(msg, fields...)
	}
}

func Fatal(msg string, fields ...zap.Field) {
	if L != nil {
		L.Fatal(msg, fields...)
	}
}

func Infof(format string, args ...interface{}) {
	if L != nil {
		L.Sugar().Infof(format, args...)
	}
}

func Warnf(format string, args ...interface{}) {
	if L != nil {
		L.Sugar().Warnf(format, args...)
	}
}

func Errorf(format string, args ...interface{}) {
	if L != nil {
		L.Sugar().Errorf(format, args...)
	}
}

func Debugf(format string, args ...interface{}) {
	if L != nil {
		L.Sugar().Debugf(format, args...)
	}
}