package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v2"
)

type logLevel string

const (
	logLevelDebug logLevel = "debug"
	logLevelInfo  logLevel = "info"
	logLevelWarn  logLevel = "warn"
	logLevelError logLevel = "error"
)

func (l logLevel) slogLevel() slog.Level {
	switch l {
	case logLevelDebug:
		return slog.LevelDebug
	case logLevelInfo:
		return slog.LevelInfo
	case logLevelWarn:
		return slog.LevelWarn
	case logLevelError:
		return slog.LevelError
	default:
		panic("unknown log level: " + l)
	}
}

func (l logLevel) String() string {
	return string(l)
}

var (
	logFlagLevel = &cli.GenericFlag{
		Name: "log.level",
		Value: &EnumValue{
			Enum: []string{
				logLevelDebug.String(),
				logLevelInfo.String(),
				logLevelWarn.String(),
				logLevelError.String(),
			},
			Default: logLevelInfo.String(),
		},
		Usage:   fmt.Sprintf("%s, %s, %s or %s", logLevelDebug, logLevelInfo, logLevelWarn, logLevelError),
		EnvVars: []string{"LOG_LEVEL"},
	}
	logFlags = []cli.Flag{
		logFlagLevel,
	}
)

func newLoggerFromFlags(c *cli.Context) *slog.Logger {
	return newLogger(loggerConfig{
		level: logLevel(c.String(logFlagLevel.Name)),
	})
}

type loggerConfig struct {
	level logLevel
}

func newLogger(cfg loggerConfig) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: cfg.level.slogLevel(),
	}))
}

type loggerCtxKey struct{}

func loggerToCtx(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, l)
}

func loggerFromCtx(ctx context.Context) *slog.Logger {
	logger, _ := ctx.Value(loggerCtxKey{}).(*slog.Logger)
	return logger
}
