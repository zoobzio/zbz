package zbz

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger is an interface exposing only the basic logging functions you need.
type Logger interface {
	Debug(args ...any)
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)
	Fatal(args ...any)
	Debugf(template string, args ...any)
	Infof(template string, args ...any)
	Warnf(template string, args ...any)
	Errorf(template string, args ...any)
	Fatalf(template string, args ...any)
}

// zLogger wraps a zap.SugaredLogger and implements the Logger interface.
type zLogger struct {
	*zap.SugaredLogger
}

// Log is a global logger instance that can be used throughout the application.
var Log Logger

// Middleware logs incoming requests and their responses using the gin context.
func LogMiddleware(c *gin.Context) {
	Log.Debugf("Request: %s %s", c.Request.Method, c.Request.URL.Path)
	c.Next()
	if len(c.Errors) > 0 {
		for _, err := range c.Errors {
			Log.Errorf("Error: %v", err)
		}
	} else {
		Log.Infof("Response: %d %s", c.Writer.Status(), c.Request.URL.Path)
	}
}

// init initializes the global logger with a zap configuration.
func init() {
	c := zap.Config{
		Encoding:         "console",
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
	}

	zp, err := c.Build()
	if err != nil {
		panic("failed to create zap logger: " + err.Error())
	}

	defer zp.Sync() // flushes buffer, if any

	Log = &zLogger{zp.Sugar()}
}
