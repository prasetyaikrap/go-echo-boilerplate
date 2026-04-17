package configurations

import (
	"context"
	"log/slog"
	"os"
)

type SlogLogger struct {
    handler *slog.Logger
}

func NewSlogLogger() *SlogLogger {
    handler := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    return &SlogLogger{handler: handler}
}

func (l *SlogLogger) Info(msg string, args ...any)  { l.handler.Info(msg, args...) }
func (l *SlogLogger) Error(msg string, args ...any) { l.handler.Error(msg, args...) }
func (l *SlogLogger) Fatal(msg string, args ...any) { l.handler.Error(msg, args...); os.Exit(1) }
func (l *SlogLogger) Warn(msg string, args ...any)  { l.handler.Warn(msg, args...) }
func (l *SlogLogger) Debug(msg string, args ...any)  { l.handler.Debug(msg, args...) }
func (l *SlogLogger) LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
    l.handler.LogAttrs(ctx, level, msg, attrs...)
}
