package closers

import (
	"io"
	"log/slog"
)

func CloseOrLog(c io.Closer, l *slog.Logger) {
	if err := c.Close(); err != nil {
		l.Error("close", "error", err)
	}
}
