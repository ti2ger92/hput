package logger

import (
	"errors"

	"go.uber.org/zap"
)

// Logger logs out.
type Logger struct {
	logger *zap.Logger
}

const (
	DebugLevel = "debug"
	InfoLevel  = "info"
	WarnLevel  = "warn"
	ErrorLevel = "error"
)

func New(level string) (Logger, error) {
	cfg := zap.NewDevelopmentConfig()
	switch level {
	case DebugLevel:
	case "", InfoLevel:
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case WarnLevel:
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case ErrorLevel:
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		return Logger{}, errors.New("Invalid level passed to logger.New")
	}

	logger, err := cfg.Build()
	if err != nil {
		return Logger{}, err
	}

	return Logger{
		logger: logger,
	}, nil
}

func (l *Logger) Infof(msg string, args ...interface{}) {
	sugar := l.logger.Sugar()
	sugar.Infof(msg, args...)
}

func (l *Logger) Debugf(msg string, args ...interface{}) {
	sugar := l.logger.Sugar()
	sugar.Debugf(msg, args...)
}

func (l *Logger) Debug(msg string) {
	l.logger.Debug(msg)
}

func (l *Logger) Warnf(msg string, args ...interface{}) {
	sugar := l.logger.Sugar()
	sugar.Warnf(msg, args...)
}

func (l *Logger) Errorf(msg string, args ...interface{}) {
	sugar := l.logger.Sugar()
	sugar.Errorf(msg, args...)
}

func (l *Logger) Sync() {
	l.logger.Sync()
}
