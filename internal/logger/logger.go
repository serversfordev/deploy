package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Logger struct {
	writers []io.Writer
}

func New(appDir string) (*Logger, error) {
	// Create log file with current date
	currentTime := time.Now()
	logFileName := fmt.Sprintf("deploy-%s.log", currentTime.Format("2006-01-02"))
	logFile, err := os.OpenFile(
		filepath.Join(appDir, "logs", logFileName),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	return &Logger{
		writers: []io.Writer{os.Stdout, logFile},
	}, nil
}

func (l *Logger) Write(p []byte) (n int, err error) {
	for _, w := range l.writers {
		n, err = w.Write(p)
		if err != nil {
			return n, err
		}
	}
	return len(p), nil
}

func (l *Logger) Printf(format string, v ...interface{}) {
	message := fmt.Sprintf(format+"\n", v...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formattedMessage := fmt.Sprintf("[%s] %s", timestamp, message)
	_, _ = l.Write([]byte(formattedMessage))
}
