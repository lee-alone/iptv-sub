package utils

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Logger 日志记录器
type Logger struct {
	*log.Logger
}

// NewLogger 创建新的日志记录器
func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", 0),
	}
}

// formatLog 格式化日志消息
func (l *Logger) formatLog(level, message string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	return fmt.Sprintf("[%s] [%s] %s", timestamp, level, message)
}

// Info 记录信息级别日志
func (l *Logger) Info(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	l.Println(l.formatLog("INFO", message))
}

// Warn 记录警告级别日志
func (l *Logger) Warn(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	l.Println(l.formatLog("WARN", message))
}

// Error 记录错误级别日志
func (l *Logger) Error(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	l.Println(l.formatLog("ERROR", message))
}

// Debug 记录调试级别日志
func (l *Logger) Debug(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	l.Println(l.formatLog("DEBUG", message))
}
