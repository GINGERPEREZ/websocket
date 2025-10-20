package shared

import "log/slog"

// Info proxies to the global slog logger to preserve compatibility with legacy helpers.
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Error proxies to the global slog logger while keeping the original signature.
func Error(err error, args ...any) {
	if err == nil {
		return
	}
	slog.Error(err.Error(), args...)
}
