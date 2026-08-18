package logx

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

// New builds a text slog logger. Accepted levels match the original config:
// debug, info, error (warn is also accepted).
func New(level string) (*slog.Logger, error) {
	return NewWriter(os.Stderr, level)
}

func NewWriter(w io.Writer, level string) (*slog.Logger, error) {
	var lvl slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		return nil, errUnsupportedLevel(level)
	}

	h := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: lvl,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format("2006-01-02 15:04:05"))
				}
			}
			return a
		},
	})
	return slog.New(h), nil
}

type unsupportedLevelError string

func errUnsupportedLevel(level string) error { return unsupportedLevelError(level) }

func (e unsupportedLevelError) Error() string {
	return "unsupported log level: " + string(e)
}
