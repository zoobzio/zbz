package zap

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"zbz/zlog"
)

// zapProvider implements ZlogProvider interface using zap
type zapProvider struct {
	logger *zap.Logger
}

// NewWithHodor creates a new zap-based contract with hodor storage support
// This is the ONLY constructor - enforces explicit hodor usage for file storage
func NewWithHodor(config Config, hodorContract *zlog.HodorContract) *zlog.ZlogContract[*zap.Logger] {
	// Validate input
	if hodorContract == nil {
		panic("zap adapter requires hodor contract for file storage - use console-only simple provider instead")
	}

	// Apply defaults using universal services
	if config.Level == "" {
		config.Level = "info"
	}
	if config.Format == "" {
		config.Format = "json"
	}
	if config.Name == "" {
		config.Name = "zap-logger"
	}

	// Parse level using universal service
	level := zlog.ParseLevel(config.Level)
	zapLevel := zlog.ConvertLevel(level).ToZapLevel().(zapcore.Level)

	// Get provider-specific format
	formatConverter := zlog.GetFormatConverter()
	zapFormat := formatConverter.GetProviderFormat(config.Format, "zap")

	// Create universal writer (console + hodor)
	keyPrefix := config.Name + "-logs"
	writer := zlog.CreateStandardWriter(config.Format, *hodorContract, keyPrefix)

	// Create zap-specific encoder
	encoder := createZapEncoder(zapFormat)

	// Create zapcore.WriteSyncer from universal writer
	syncer := zapcore.AddSync(writer)

	// Create single core with universal writer
	core := zapcore.NewCore(encoder, syncer, zapLevel)

	// Apply sampling if configured (universal sampling could be added later)
	if config.Sampling != nil {
		core = zapcore.NewSamplerWithOptions(
			core,
			time.Second, // Sample per second
			config.Sampling.Initial,
			config.Sampling.Thereafter,
		)
	}

	// Create zap logger
	zapLog := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(3), // Skip: zapProvider.Info() -> zZlogService.Info() -> zlog.Info() -> user code
		zap.AddStacktrace(zap.ErrorLevel))

	// Create provider
	provider := &zapProvider{
		logger: zapLog,
	}

	// Return contract with both service and typed logger
	return zlog.NewContract[*zap.Logger](config.Name, provider, zapLog)
}

// Info logs at info level
func (z *zapProvider) Info(msg string, fields []zlog.Field) {
	zapFields := convertFields(fields)
	z.logger.Info(msg, zapFields...)
}

// Error logs at error level
func (z *zapProvider) Error(msg string, fields []zlog.Field) {
	zapFields := convertFields(fields)
	z.logger.Error(msg, zapFields...)
}

// Debug logs at debug level
func (z *zapProvider) Debug(msg string, fields []zlog.Field) {
	zapFields := convertFields(fields)
	z.logger.Debug(msg, zapFields...)
}

// Warn logs at warn level
func (z *zapProvider) Warn(msg string, fields []zlog.Field) {
	zapFields := convertFields(fields)
	z.logger.Warn(msg, zapFields...)
}

// Fatal logs at fatal level and exits
func (z *zapProvider) Fatal(msg string, fields []zlog.Field) {
	zapFields := convertFields(fields)
	z.logger.Fatal(msg, zapFields...)
}

// Close cleans up the provider
func (z *zapProvider) Close() error {
	return z.logger.Sync()
}

// convertFields converts zlog fields to zap fields (simplified - could be universalized)
func convertFields(fields []zlog.Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		zapFields[i] = convertField(f)
	}
	return zapFields
}

// convertField converts a single zlog field to zap field (could be universalized)
func convertField(f zlog.Field) zap.Field {
	switch f.Type {
	case zlog.StringType:
		return zap.String(f.Key, f.Value.(string))
	case zlog.IntType:
		return zap.Int(f.Key, f.Value.(int))
	case zlog.Int64Type:
		return zap.Int64(f.Key, f.Value.(int64))
	case zlog.Float64Type:
		return zap.Float64(f.Key, f.Value.(float64))
	case zlog.BoolType:
		return zap.Bool(f.Key, f.Value.(bool))
	case zlog.ErrorType:
		return zap.Error(f.Value.(error))
	case zlog.DurationType:
		return zap.Duration(f.Key, f.Value.(time.Duration))
	case zlog.TimeType:
		return zap.Time(f.Key, f.Value.(time.Time))
	case zlog.ByteStringType:
		return zap.String(f.Key, f.Value.(string))
	case zlog.AnyType:
		return zap.Any(f.Key, f.Value)
	case zlog.StringsType:
		return zap.Strings(f.Key, f.Value.([]string))
	default:
		return zap.Any(f.Key, f.Value)
	}
}

// createZapEncoder creates appropriate zap encoder (simplified)
func createZapEncoder(format string) zapcore.Encoder {
	switch format {
	case "console":
		config := zap.NewDevelopmentEncoderConfig()
		config.EncodeTime = zapcore.ISO8601TimeEncoder
		return zapcore.NewConsoleEncoder(config)
	case "json":
		return zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	default:
		return zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	}
}