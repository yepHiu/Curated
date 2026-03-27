package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// encodeTimeBracketed formats log lines as [日期][时间]-… (console encoder writes level/message after the time field).
func encodeTimeBracketed(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + t.Format("2006-01-02") + "][" + t.Format("15:04:05") + "]-")
}

func encoderConfig() zapcore.EncoderConfig {
	ec := zap.NewProductionEncoderConfig()
	ec.EncodeTime = encodeTimeBracketed
	ec.EncodeLevel = zapcore.CapitalLevelEncoder
	return ec
}

// FileSink configures optional daily rotated log files under LogDir. Zero value disables file output.
type FileSink struct {
	LogDir     string
	FilePrefix string
	MaxAgeDays int
}

// New builds a logger that writes to stdout. If sink.LogDir is non-empty, the same encoder also writes to daily rotated files (tee).
func New(level string, sink FileSink) (*zap.Logger, error) {
	if level == "" {
		level = "info"
	}
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		return nil, err
	}
	at := zap.NewAtomicLevelAt(zapLevel)

	ec := encoderConfig()
	consoleCore := zapcore.NewCore(zapcore.NewConsoleEncoder(ec), zapcore.AddSync(os.Stdout), at)

	cores := []zapcore.Core{consoleCore}

	if dir := strings.TrimSpace(sink.LogDir); dir != "" {
		prefix := strings.TrimSpace(sink.FilePrefix)
		if prefix == "" {
			prefix = "javd"
		}
		maxAgeDays := sink.MaxAgeDays
		if maxAgeDays <= 0 {
			maxAgeDays = 7
		}

		absDir, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("log dir abs: %w", err)
		}
		if err := os.MkdirAll(absDir, 0o755); err != nil {
			return nil, fmt.Errorf("create log dir: %w", err)
		}

		// strftime placeholders in filename (see github.com/lestrrat-go/file-rotatelogs)
		pattern := filepath.Join(absDir, prefix+"-%Y%m%d.log")
		w, err := rotatelogs.New(
			pattern,
			rotatelogs.WithMaxAge(time.Duration(maxAgeDays)*24*time.Hour),
			rotatelogs.WithRotationTime(24*time.Hour),
		)
		if err != nil {
			return nil, fmt.Errorf("rotating log file: %w", err)
		}

		fileCore := zapcore.NewCore(zapcore.NewConsoleEncoder(ec), zapcore.AddSync(w), at)
		cores = append(cores, fileCore)
	}

	return zap.New(
		zapcore.NewTee(cores...),
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	), nil
}

func Error(err error) zap.Field {
	return zap.Error(err)
}
