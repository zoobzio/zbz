package zbz

import (
	"encoding/json"
	"maps"
	//"strings"
	"sync"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

type safeSyncer struct {
	zapcore.WriteSyncer
	mu sync.Mutex
}

func (s *safeSyncer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.WriteSyncer.Write(p)
}

type PrettyConsoleEncoder struct {
	zapcore.Encoder
	cfg zapcore.EncoderConfig
}

var levelColors = map[zapcore.Level]string{
	zapcore.DebugLevel:  "\033[34m",   // Blue
	zapcore.InfoLevel:   "\033[32m",   // Green
	zapcore.WarnLevel:   "\033[33m",   // Yellow
	zapcore.ErrorLevel:  "\033[31m",   // Red
	zapcore.DPanicLevel: "\033[35m",   // Magenta
	zapcore.PanicLevel:  "\033[35m",   // Magenta
	zapcore.FatalLevel:  "\033[1;31m", // Bold Red
}

const (
	gray       = "\033[38;5;250m"
	colorReset = "\033[0m"
)

func NewPrettyConsoleEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	return &PrettyConsoleEncoder{
		Encoder: zapcore.NewConsoleEncoder(cfg),
		cfg:     cfg,
	}
}

func (e *PrettyConsoleEncoder) Clone() zapcore.Encoder {
	return &PrettyConsoleEncoder{
		Encoder: e.Encoder.Clone(),
		cfg:     e.cfg,
	}
}

func (e *PrettyConsoleEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	header, err := e.Encoder.EncodeEntry(entry, nil)
	if err != nil {
		return nil, err
	}

	// Collect fields into a map
	fieldMap := make(map[string]any)
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fields {
		f.AddTo(enc)
	}
	maps.Copy(fieldMap, enc.Fields)

	// Add the level color to the header
	color := levelColors[entry.Level]
	headerStr := header.String()
	if len(headerStr) > 0 && headerStr[len(headerStr)-1] == '\n' {
		headerStr = headerStr[:len(headerStr)-1]
	}
	coloredHeader := color + headerStr + colorReset

	header.Reset()
	header.AppendString(coloredHeader + "\n")

	// Only pretty-print and append if there are fields
	if len(fieldMap) > 0 {
		jsonBytes, err := json.Marshal(fieldMap)
		if err != nil {
			return nil, err
		}
		jsonStr := string(jsonBytes)

		/*
			      separator := "  │ "
						jsonLines := strings.Split(jsonStr, "\n")
						var block strings.Builder
						for _, line := range jsonLines {
							if line != "" {
								block.WriteString(separator)
								block.WriteString(line)
								block.WriteByte('\n')
							}
						}
		*/

		header.AppendString(gray)
		header.AppendString(jsonStr)
		header.AppendString("\n")
		header.AppendString(colorReset)
	}

	return header, nil
}
