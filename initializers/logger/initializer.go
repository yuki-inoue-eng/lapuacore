package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"
)

// InitLogger sets up file logging if the LAPUA_FILE_LOGGING environment
// variable is set to "true". The log file is closed after ctx is cancelled.
func InitLogger(ctx context.Context, logFilePath string) io.Closer {
	if os.Getenv("LAPUA_FILE_LOGGING") != "true" {
		return nil
	}

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	go func() {
		defer logFile.Close()
		for {
			select {
			case <-ctx.Done():
				time.Sleep(1 * time.Second)
				return
			}
		}
	}()

	return logFile
}
