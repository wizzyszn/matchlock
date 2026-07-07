package logger

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config controls Zap logger construction.
type Config struct {
	Level    string
	Encoding string
}

// Configure installs Zap as the process logger and routes slog events through it.
func Configure(cfg Config) (*zap.Logger, error) {
	zapLogger, err := New(cfg)
	if err != nil {
		return nil, err
	}
	zap.ReplaceGlobals(zapLogger)
	slog.SetDefault(slog.New(NewSlogHandler(zapLogger)))
	return zapLogger, nil
}

// New builds a production-oriented Zap logger.
func New(cfg Config) (*zap.Logger, error) {
	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	if strings.TrimSpace(cfg.Level) != "" {
		if err := level.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(cfg.Level)))); err != nil {
			return nil, fmt.Errorf("invalid log level %q: %w", cfg.Level, err)
		}
	}

	encoding := strings.ToLower(strings.TrimSpace(cfg.Encoding))
	if encoding == "" {
		encoding = "json"
	}
	if encoding != "json" && encoding != "console" {
		return nil, fmt.Errorf("invalid log encoding %q (want json or console)", cfg.Encoding)
	}

	zapCfg := zap.NewProductionConfig()
	zapCfg.Level = level
	zapCfg.Encoding = encoding
	zapCfg.EncoderConfig.TimeKey = "ts"
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapCfg.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	zapCfg.InitialFields = map[string]any{"service": "matchlock-backend"}
	if encoding == "console" {
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	return zapCfg.Build()
}

type slogHandler struct {
	logger *zap.Logger
	attrs  []zap.Field
	groups []string
}

// NewSlogHandler returns a slog handler that writes records through Zap.
func NewSlogHandler(logger *zap.Logger) slog.Handler {
	if logger == nil {
		logger = zap.L()
	}
	return &slogHandler{logger: logger}
}

func (h *slogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h.logger.Core().Enabled(zapLevel(level))
}

func (h *slogHandler) Handle(_ context.Context, record slog.Record) error {
	fields := make([]zap.Field, 0, len(h.attrs)+record.NumAttrs()+2)
	fields = append(fields, h.attrs...)
	if !record.Time.IsZero() {
		fields = append(fields, zap.Time("slog_time", record.Time))
	}
	if record.PC != 0 {
		frame, _ := runtime.CallersFrames([]uintptr{record.PC}).Next()
		fields = append(fields, zap.String("source", fmt.Sprintf("%s:%d", frame.File, frame.Line)))
	}
	record.Attrs(func(attr slog.Attr) bool {
		fields = append(fields, slogAttrToField(h.key(attr.Key), attr.Value))
		return true
	})

	if checked := h.logger.Check(zapLevel(record.Level), record.Message); checked != nil {
		checked.Write(fields...)
	}
	return nil
}

func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := h.clone()
	for _, attr := range attrs {
		next.attrs = append(next.attrs, slogAttrToField(next.key(attr.Key), attr.Value))
	}
	return next
}

func (h *slogHandler) WithGroup(name string) slog.Handler {
	if strings.TrimSpace(name) == "" {
		return h
	}
	next := h.clone()
	next.groups = append(next.groups, name)
	return next
}

func (h *slogHandler) clone() *slogHandler {
	return &slogHandler{
		logger: h.logger,
		attrs:  append([]zap.Field(nil), h.attrs...),
		groups: append([]string(nil), h.groups...),
	}
}

func (h *slogHandler) key(key string) string {
	if len(h.groups) == 0 {
		return key
	}
	parts := make([]string, 0, len(h.groups)+1)
	parts = append(parts, h.groups...)
	parts = append(parts, key)
	return strings.Join(parts, ".")
}

func zapLevel(level slog.Level) zapcore.Level {
	switch {
	case level >= slog.LevelError:
		return zapcore.ErrorLevel
	case level >= slog.LevelWarn:
		return zapcore.WarnLevel
	case level <= slog.LevelDebug:
		return zapcore.DebugLevel
	default:
		return zapcore.InfoLevel
	}
}

func slogAttrToField(key string, value slog.Value) zap.Field {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		return zap.String(key, value.String())
	case slog.KindBool:
		return zap.Bool(key, value.Bool())
	case slog.KindInt64:
		return zap.Int64(key, value.Int64())
	case slog.KindUint64:
		return zap.Uint64(key, value.Uint64())
	case slog.KindFloat64:
		return zap.Float64(key, value.Float64())
	case slog.KindDuration:
		return zap.Duration(key, value.Duration())
	case slog.KindTime:
		return zap.Time(key, value.Time())
	case slog.KindGroup:
		return zap.Any(key, slogGroup(value.Group()))
	default:
		return zap.Any(key, value.Any())
	}
}

func slogGroup(attrs []slog.Attr) map[string]any {
	group := make(map[string]any, len(attrs))
	for _, attr := range attrs {
		value := attr.Value.Resolve()
		switch value.Kind() {
		case slog.KindString:
			group[attr.Key] = value.String()
		case slog.KindBool:
			group[attr.Key] = value.Bool()
		case slog.KindInt64:
			group[attr.Key] = value.Int64()
		case slog.KindUint64:
			group[attr.Key] = value.Uint64()
		case slog.KindFloat64:
			group[attr.Key] = value.Float64()
		case slog.KindDuration:
			group[attr.Key] = value.Duration()
		case slog.KindTime:
			group[attr.Key] = value.Time().Format(time.RFC3339Nano)
		default:
			group[attr.Key] = value.Any()
		}
	}
	return group
}
