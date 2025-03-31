package mocks

import (
	"bytes"
	"log/slog"
)

func NewLoggerMock() (*bytes.Buffer, *slog.Logger) {
	buf := &bytes.Buffer{}
	return buf, slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
}
