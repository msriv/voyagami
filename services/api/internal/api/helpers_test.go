package api_test

import "log/slog"

func nilLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }
