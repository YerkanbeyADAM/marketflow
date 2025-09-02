package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

type CustomLogger struct {
	infoLogger *log.Logger
	warnLogger *log.Logger
	errLogger  *log.Logger
	mu         sync.RWMutex
}

func NewCustomLogger() (*CustomLogger, error) {
	flags := log.Ldate | log.Ltime | log.Lshortfile

	err := os.MkdirAll("logs", 0o755)
	if err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Открываем файлы логов
	fileInfo, err := os.OpenFile("logs/info.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		return nil, fmt.Errorf("failed to open info log file: %w", err)
	}
	fileWarn, err := os.OpenFile("logs/warning.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		return nil, fmt.Errorf("failed to open warning log file: %w", err)
	}
	fileErr, err := os.OpenFile("logs/error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		return nil, fmt.Errorf("failed to open error log file: %w", err)
	}

	// Настраиваем MultiWriter: файл + stdout
	infoWriter := io.MultiWriter(os.Stdout, fileInfo)
	warnWriter := io.MultiWriter(os.Stdout, fileWarn)
	errWriter := io.MultiWriter(os.Stderr, fileErr) // ошибки в stderr

	logger := &CustomLogger{
		infoLogger: log.New(infoWriter, "INFO: ", flags),
		warnLogger: log.New(warnWriter, "WARN: ", flags),
		errLogger:  log.New(errWriter, "ERROR: ", flags),
	}

	return logger, nil
}

func (l *CustomLogger) Info(msg ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.infoLogger.Println(msg...)
}

func (l *CustomLogger) Warn(msg ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warnLogger.Println(msg...)
}

func (l *CustomLogger) Error(msg ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errLogger.Println(msg...)
}
