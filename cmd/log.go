package cmd

import (
	"io"
	"log/slog"

	"github.com/lmittmann/tint"
)

func setLogger(output io.Writer) {
	logLevel := &slog.LevelVar{}
	opts := &tint.Options{
		Level:     logLevel,
		AddSource: false,
	}

	logLevel.Set(slog.LevelDebug)

	handler := tint.NewHandler(output, opts)
	logger := slog.New(handler)

	slog.SetDefault(logger)
}
