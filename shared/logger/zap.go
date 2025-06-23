package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// zapLogger implements Logger interface using zap
type zapLogger struct {
	logger *zap.Logger
}

// NewZapLogger creates a new zap-based logger with file rotation and console output
func NewZapLogger() Logger {
	devMode := os.Getenv("ZBZ_ENV") == "development"
	logFile := os.Getenv("ZBZ_LOG_FILE")
	if logFile == "" {
		logFile = ".logs/app.log"
	}

	var cores []zapcore.Core

	// Always add console output (essential for containers)
	var consoleEncoder zapcore.Encoder
	if devMode {
		// Human-readable format for development
		consoleConfig := zap.NewDevelopmentEncoderConfig()
		consoleConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		consoleEncoder = zapcore.NewConsoleEncoder(consoleConfig)
	} else {
		// JSON format for production (structured logs)
		consoleEncoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	}
	consoleSyncer := zapcore.AddSync(os.Stdout)
	cores = append(cores, zapcore.NewCore(consoleEncoder, consoleSyncer, zap.DebugLevel))

	// File output with lumberjack rotation (additional persistence)
	fileSyncer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    10, // megabytes
		MaxBackups: 5,
		MaxAge:     7, // days
		Compress:   true,
	})
	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	cores = append(cores, zapcore.NewCore(fileEncoder, fileSyncer, zap.DebugLevel))

	core := zapcore.NewTee(cores...)
	zapLog := zap.New(core, 
		zap.AddCaller(), 
		zap.AddCallerSkip(3), // Skip: zap call -> zapLogger method -> global wrapper
		zap.AddStacktrace(zap.ErrorLevel))

	return &zapLogger{logger: zapLog}
}

// convertFields converts our Field types to zap.Field types
func (z *zapLogger) convertFields(fields []Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		zapFields[i] = z.convertField(f)
	}
	return zapFields
}

// convertField converts a single Field to zap.Field
func (z *zapLogger) convertField(f Field) zap.Field {
	switch f.Type {
	case StringType:
		return zap.String(f.Key, f.Value.(string))
	case IntType:
		return zap.Int(f.Key, f.Value.(int))
	case Int64Type:
		return zap.Int64(f.Key, f.Value.(int64))
	case Float64Type:
		return zap.Float64(f.Key, f.Value.(float64))
	case BoolType:
		return zap.Bool(f.Key, f.Value.(bool))
	case ErrorType:
		return zap.Error(f.Value.(error))
	case DurationType:
		return zap.Duration(f.Key, f.Value.(time.Duration))
	case TimeType:
		return zap.Time(f.Key, f.Value.(time.Time))
	case ByteStringType:
		return zap.String(f.Key, f.Value.(string))
	case AnyType:
		return zap.Any(f.Key, f.Value)
	case StringsType:
		return zap.Strings(f.Key, f.Value.([]string))
	default:
		// Fallback to Any for unknown types
		return zap.Any(f.Key, f.Value)
	}
}

func (z *zapLogger) Info(msg string, fields ...Field) {
	z.logger.Info(msg, z.convertFields(fields)...)
}

func (z *zapLogger) Error(msg string, fields ...Field) {
	z.logger.Error(msg, z.convertFields(fields)...)
}

func (z *zapLogger) Debug(msg string, fields ...Field) {
	z.logger.Debug(msg, z.convertFields(fields)...)
}

func (z *zapLogger) Warn(msg string, fields ...Field) {
	z.logger.Warn(msg, z.convertFields(fields)...)
}

func (z *zapLogger) Fatal(msg string, fields ...Field) {
	z.logger.Fatal(msg, z.convertFields(fields)...)
}