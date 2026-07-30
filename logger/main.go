package logger

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"
)

const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	white   = "\033[37m"
)

const (
	ColourApp     = cyan
	ColourServer  = blue
	ColourRequest = green
)

type Logger struct {
	module string
	colour string
	l      *log.Logger
}

func New(module, colour string) *Logger {
	return &Logger{
		module: module,
		colour: colour,
		l:      log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) prefix(level, levelColour string) string {
	ts := time.Now().UTC().Format(time.RFC3339)
	mod := fmt.Sprintf("%s%s[%s]%s", bold, l.colour, l.module, reset)
	lvl := fmt.Sprintf("%s%s[%s]%s", bold, levelColour, level, reset)
	return ts + " " + mod + " " + lvl
}

func (l *Logger) Info(v ...any) {
	l.l.Print(l.prefix("INFO", white) + " " + fmt.Sprint(v...))
}

func (l *Logger) Infof(format string, v ...any) {
	l.l.Print(l.prefix("INFO", white) + " " + fmt.Sprintf(format, v...))
}

func (l *Logger) Warn(v ...any) {
	l.l.Print(l.prefix("WARN", yellow) + " " + fmt.Sprint(v...))
}

func (l *Logger) Warnf(format string, v ...any) {
	l.l.Print(l.prefix("WARN", yellow) + " " + fmt.Sprintf(format, v...))
}

func (l *Logger) Error(v ...any) {
	l.l.Print(l.prefix("ERROR", red) + " " + fmt.Sprint(v...))
}

func (l *Logger) Errorf(format string, v ...any) {
	l.l.Print(l.prefix("ERROR", red) + " " + fmt.Sprintf(format, v...))
}

func (l *Logger) Debug(v ...any) {
	l.l.Print(l.prefix("DEBUG", magenta) + " " + fmt.Sprint(v...))
}

func (l *Logger) Debugf(format string, v ...any) {
	l.l.Print(l.prefix("DEBUG", magenta) + " " + fmt.Sprintf(format, v...))
}

func (l *Logger) Fatal(v ...any) {
	l.l.Fatal(l.prefix("FATAL", red) + " " + fmt.Sprint(v...))
}

func (l *Logger) Fatalf(format string, v ...any) {
	l.l.Fatal(l.prefix("FATAL", red) + " " + fmt.Sprintf(format, v...))
}

type Handler struct {
	module string
	colour string
	l      *log.Logger
}

func NewHandler(module, colour string) *Handler {
	return &Handler{
		module: module,
		colour: colour,
		l:      log.New(os.Stdout, "", 0),
	}
}

func (h *Handler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	levelColour := white
	switch r.Level {
	case slog.LevelDebug:
		levelColour = magenta
	case slog.LevelWarn:
		levelColour = yellow
	case slog.LevelError:
		levelColour = red
	}
	t := r.Time
	if t.IsZero() {
		t = time.Now()
	}

	ts := t.UTC().Format(time.RFC3339)

	mod := fmt.Sprintf("%s%s[%s]%s", bold, h.colour, h.module, reset)
	lvl := fmt.Sprintf("%s%s[%s]%s", bold, levelColour, r.Level.String(), reset)

	var b strings.Builder
	b.WriteString(ts)
	b.WriteByte(' ')
	b.WriteString(mod)
	b.WriteByte(' ')
	b.WriteString(lvl)
	b.WriteByte(' ')
	b.WriteString(r.Message)

	r.Attrs(func(a slog.Attr) bool {
		b.WriteByte(' ')
		b.WriteString(a.Key)
		b.WriteByte('=')
		b.WriteString(a.Value.String())
		return true
	})

	h.l.Print(b.String())
	return nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *Handler) WithGroup(name string) slog.Handler       { return h }
