package yggdrasil

import (
	"fmt"

	"go.uber.org/zap"
)

// CoreLogger adapts zap.Logger to Yggdrasil core.Logger interface
type CoreLogger struct {
	logger *zap.SugaredLogger
}

// NewCoreLogger creates a new CoreLogger
func NewCoreLogger(logger *zap.SugaredLogger) *CoreLogger {
	return &CoreLogger{logger: logger}
}

// Printf implements core.Logger interface
func (l *CoreLogger) Printf(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

// Println implements core.Logger interface
func (l *CoreLogger) Println(args ...interface{}) {
	l.logger.Info(args...)
}

// Debugf logs debug messages
func (l *CoreLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

// Infof logs info messages
func (l *CoreLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

// Warnf logs warning messages
func (l *CoreLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

// Errorf logs error messages
func (l *CoreLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

// MulticastLogger adapts zap.Logger to multicast logger interface
type MulticastLogger struct {
	logger *zap.SugaredLogger
}

// NewMulticastLogger creates a new MulticastLogger
func NewMulticastLogger(logger *zap.SugaredLogger) *MulticastLogger {
	return &MulticastLogger{logger: logger}
}

// Printf implements log.Logger interface
func (l *MulticastLogger) Printf(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

// Print implements log.Logger interface
func (l *MulticastLogger) Print(args ...interface{}) {
	l.logger.Info(args...)
}

// Println implements log.Logger interface
func (l *MulticastLogger) Println(args ...interface{}) {
	l.logger.Info(fmt.Sprint(args...))
}

// Fatal implements log.Logger interface
func (l *MulticastLogger) Fatal(args ...interface{}) {
	l.logger.Fatal(args...)
}

// Fatalf implements log.Logger interface
func (l *MulticastLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}

// Fatalln implements log.Logger interface
func (l *MulticastLogger) Fatalln(args ...interface{}) {
	l.logger.Fatal(fmt.Sprint(args...))
}

// Panic implements log.Logger interface
func (l *MulticastLogger) Panic(args ...interface{}) {
	l.logger.Panic(args...)
}

// Panicf implements log.Logger interface
func (l *MulticastLogger) Panicf(format string, args ...interface{}) {
	l.logger.Panicf(format, args...)
}

// Panicln implements log.Logger interface
func (l *MulticastLogger) Panicln(args ...interface{}) {
	l.logger.Panic(fmt.Sprint(args...))
}
