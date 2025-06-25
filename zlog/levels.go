package zlog

// LevelConverter provides conversion from universal LogLevel to provider-specific levels
type LevelConverter struct {
	level LogLevel
}

// NewLevelConverter creates a level converter for the given universal level
func NewLevelConverter(level LogLevel) *LevelConverter {
	return &LevelConverter{level: level}
}

// ToZapLevel converts universal level to zap level (interface{} to avoid import)
func (l *LevelConverter) ToZapLevel() interface{} {
	// Return the level value that zap expects
	// We use interface{} to avoid importing zap in the universal package
	switch l.level {
	case DEBUG:
		return -1 // zap.DebugLevel
	case INFO:
		return 0  // zap.InfoLevel
	case WARN:
		return 1  // zap.WarnLevel
	case ERROR:
		return 2  // zap.ErrorLevel
	case FATAL:
		return 5  // zap.FatalLevel
	default:
		return 0  // Default to info
	}
}

// ToZerologLevel converts universal level to zerolog level (interface{} to avoid import)
func (l *LevelConverter) ToZerologLevel() interface{} {
	// Return the level value that zerolog expects
	switch l.level {
	case DEBUG:
		return -1 // zerolog.DebugLevel
	case INFO:
		return 0  // zerolog.InfoLevel
	case WARN:
		return 1  // zerolog.WarnLevel
	case ERROR:
		return 2  // zerolog.ErrorLevel
	case FATAL:
		return 5  // zerolog.FatalLevel
	default:
		return 0  // Default to info
	}
}

// ToLogrusLevel converts universal level to logrus level (interface{} to avoid import)
func (l *LevelConverter) ToLogrusLevel() interface{} {
	// Return the level value that logrus expects
	switch l.level {
	case DEBUG:
		return 5 // logrus.DebugLevel
	case INFO:
		return 4 // logrus.InfoLevel
	case WARN:
		return 3 // logrus.WarnLevel
	case ERROR:
		return 2 // logrus.ErrorLevel
	case FATAL:
		return 1 // logrus.FatalLevel
	default:
		return 4 // Default to info
	}
}

// ToSlogLevel converts universal level to slog level (interface{} to avoid import)
func (l *LevelConverter) ToSlogLevel() interface{} {
	// Return the level value that slog expects
	switch l.level {
	case DEBUG:
		return -4 // slog.LevelDebug
	case INFO:
		return 0  // slog.LevelInfo
	case WARN:
		return 4  // slog.LevelWarn
	case ERROR:
		return 8  // slog.LevelError
	case FATAL:
		return 12 // Treat as error level (slog doesn't have fatal)
	default:
		return 0  // Default to info
	}
}

// ToApexLevel converts universal level to apex level (interface{} to avoid import)
func (l *LevelConverter) ToApexLevel() interface{} {
	// Return the level value that apex expects
	switch l.level {
	case DEBUG:
		return 0 // log.DebugLevel
	case INFO:
		return 1 // log.InfoLevel
	case WARN:
		return 2 // log.WarnLevel
	case ERROR:
		return 3 // log.ErrorLevel
	case FATAL:
		return 4 // log.FatalLevel
	default:
		return 1 // Default to info
	}
}

// IsEnabled checks if a log level should be processed based on minimum level
func (l *LevelConverter) IsEnabled(minLevel LogLevel) bool {
	return l.level >= minLevel
}

// String returns the string representation for provider-agnostic use
func (l *LevelConverter) String() string {
	return l.level.String()
}

// LevelFromString creates a LevelConverter from string
func LevelFromString(level string) *LevelConverter {
	levelMgr := NewLevelManager()
	return NewLevelConverter(levelMgr.ParseLevel(level))
}

// GetAllLevels returns all supported levels in order
func GetAllLevels() []LogLevel {
	return []LogLevel{DEBUG, INFO, WARN, ERROR, FATAL}
}

// GetLevelNames returns all level names
func GetLevelNames() []string {
	return []string{"debug", "info", "warn", "error", "fatal"}
}