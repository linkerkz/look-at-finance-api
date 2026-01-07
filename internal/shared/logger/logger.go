package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.SugaredLogger
}

func New() *Logger {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = "timestamp"
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncodeLevel = zapcore.CapitalLevelEncoder

	var encoder zapcore.Encoder
	if os.Getenv("LOG_FORMAT") == "console" {
		encoder = zapcore.NewConsoleEncoder(config)
	} else {
		encoder = zapcore.NewJSONEncoder(config)
	}

	level := zapcore.InfoLevel
	if os.Getenv("LOG_LEVEL") == "debug" {
		level = zapcore.DebugLevel
	}

	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &Logger{logger.Sugar()}
}

func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{l.With("request_id", requestID)}
}

func (l *Logger) WithUserID(userID string) *Logger {
	return &Logger{l.With("user_id", userID)}
}

func (l *Logger) Fatal(msg string, err error) {
	l.SugaredLogger.Fatalw(msg, "error", err)
}

