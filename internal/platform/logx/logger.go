package logx

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates an application logger.
//
// Default format is pretty console with colors and symbols.
// Set LOG_FORMAT=json for machine-readable output.
func New() (*zap.Logger, error) {
	format := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT")))
	if format == "json" {
		cfg := zap.NewProductionConfig()
		cfg.OutputPaths = []string{"stdout"}
		cfg.ErrorOutputPaths = []string{"stderr"}
		return cfg.Build()
	}

	encCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    encodeLevel,
		EncodeTime:     zapcore.TimeEncoderOfLayout("15:04:05"),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encCfg),
		zapcore.Lock(os.Stdout),
		zap.NewAtomicLevelAt(zap.InfoLevel),
	)

	return zap.New(core, zap.AddCaller()), nil
}

func encodeLevel(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var symbol string
	var color string
	switch level {
	case zapcore.DebugLevel:
		symbol = "[.]"
		color = "\x1b[37m"
	case zapcore.InfoLevel:
		symbol = "[i]"
		color = "\x1b[32m"
	case zapcore.WarnLevel:
		symbol = "[!]"
		color = "\x1b[33m"
	case zapcore.ErrorLevel:
		symbol = "[x]"
		color = "\x1b[31m"
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		symbol = "[X]"
		color = "\x1b[35m"
	default:
		symbol = "[?]"
		color = "\x1b[36m"
	}

	label := strings.ToUpper(level.String())
	enc.AppendString(fmt.Sprintf("%s%s %s\x1b[0m", color, symbol, label))
}
