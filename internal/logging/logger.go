package logging

import "log/slog"

type Logger struct {
	name string
	base *slog.Logger
}


func New(base *slog.Logger, name string) *Logger {
	return &Logger{name: name, base: base}
}

func (l *Logger) Debug(msg string, args ...any) { l.base.Debug(l.prefix(msg), args...) }
func (l *Logger) Info(msg string, args ...any)  { l.base.Info(l.prefix(msg), args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.base.Warn(l.prefix(msg), args...) }
func (l *Logger) Error(msg string, args ...any) { l.base.Error(l.prefix(msg), args...) }

func (l *Logger) prefix(msg string) string {
	return "[" + l.name + "]: " + msg
}