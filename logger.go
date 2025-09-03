package workerpool

import (
	"context"
	"fmt"
)

type stdLogger struct{}

var _ Logger = &stdLogger{}

func NewStdLogger() *stdLogger {
	return &stdLogger{}
}

func (l *stdLogger) Info(ctx context.Context, format string, args ...interface{}) {
	fmt.Printf("INFO - "+format, args...)
}

func (l *stdLogger) Warn(ctx context.Context, format string, args ...interface{}) {
	fmt.Printf("WARN - "+format, args...)
}

type nopLogger struct{}

func (l *nopLogger) Info(ctx context.Context, format string, args ...interface{}) {
	// No operation logger does nothing
}

func (l *nopLogger) Warn(ctx context.Context, format string, args ...interface{}) {
	// No operation logger does nothing
}

func NewNopLogger() *nopLogger {
	return &nopLogger{}
}
