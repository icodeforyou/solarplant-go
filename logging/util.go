package logging

import (
	"log/slog"
	"strings"
)

func LevelFromString(str *string) slog.Level {
	if str == nil {
		return slog.LevelInfo
	}
	switch strings.ToUpper(*str) {
	case slog.LevelDebug.String():
		return slog.LevelDebug
	case slog.LevelInfo.String():
		return slog.LevelInfo
	case slog.LevelWarn.String():
		return slog.LevelWarn
	case slog.LevelError.String():
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
