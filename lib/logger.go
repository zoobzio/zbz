package zbz

import (
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
}

// zLogger implements the Logger interface using zap for structured logging.
type zLogger struct {
	*zap.Logger
}

// Log is a global logger instance that can be used throughout the application.
var Log Logger

// Middleware logs incoming requests and their responses using the gin context.
func LogMiddleware(c *gin.Context) {
	Log.Info("HTTP Request",
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("ip", c.ClientIP()),
	)
	c.Next()
	Log.Info("HTTP Response",
		zap.Int("status", c.Writer.Status()),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("ip", c.ClientIP()),
	)
}

// InitLogger initializes the global logger with zap, supporting file and console outputs.
func InitLogger(devMode bool, logFile string) {
	var cores []zapcore.Core

	// Lumberjack logger for file output with smaller rotation for dev
	fileSyncer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    10, // megabytes (smaller for dev)
		MaxBackups: 5,  // Keep more backups for debugging
		MaxAge:     7,  // days (shorter for dev)
		Compress:   true,
	})
	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	cores = append(cores, zapcore.NewCore(fileEncoder, fileSyncer, zap.DebugLevel))

	if devMode {
		// Console output (stdout/stderr)
		consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
		consoleSyncer := zapcore.AddSync(os.Stdout)
		cores = append(cores, zapcore.NewCore(consoleEncoder, consoleSyncer, zap.DebugLevel))
	}

	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	Log = &zLogger{logger}
}

func init() {
	InitLogger(true, ".logs/app.log")
}
