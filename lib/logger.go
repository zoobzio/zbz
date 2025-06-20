package zbz

import (
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger is an interface exposing only the basic logging functions you need.
type Logger interface {
	Debug(args ...any)
	Debugw(msg string, args ...any)
	Info(args ...any)
	Infow(msg string, args ...any)
	Warn(args ...any)
	Warnw(msg string, args ...any)
	Error(args ...any)
	Errorw(msg string, args ...any)
	Fatal(args ...any)
	Fatalw(msg string, args ...any)
}

// zLogger wraps a zap.SugaredLogger and implements the Logger interface.
type zLogger struct {
	*zap.SugaredLogger
}

// Log is a global logger instance that can be used throughout the application.
var Log Logger

// Middleware logs incoming requests and their responses using the gin context.
func LogMiddleware(c *gin.Context) {
	Log.Infow("HTTP Request")
	c.Next()
	Log.Infow("HTTP Response")
}

// InitLogger initializes the global logger with zap, supporting file and console outputs.
func InitLogger(devMode bool, logFile string) {
	var cores []zapcore.Core

	// Lumberjack logger for file output
	fileSyncer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    100, // megabytes
		MaxBackups: 3,
		MaxAge:     30, // days
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
	Log = &zLogger{logger.Sugar()}
}

func init() {
	InitLogger(true, ".logs/app.log")
}
