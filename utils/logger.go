package utils

import (
	"fmt"
	"log"
	"os"
	"sync"
)

// LogLevel 日志级别类型
type LogLevel int

const (
	// DEBUG 调试级别
	DEBUG LogLevel = iota
	// INFO 信息级别
	INFO
	// WARN 警告级别
	WARN
	// ERROR 错误级别
	ERROR
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel 从字符串解析日志级别
func ParseLevel(level string) LogLevel {
	switch level {
	case "debug", "DEBUG":
		return DEBUG
	case "info", "INFO":
		return INFO
	case "warn", "WARN":
		return WARN
	case "error", "ERROR":
		return ERROR
	default:
		return INFO
	}
}

var (
	globalLevel   LogLevel = INFO
	globalLevelMu sync.RWMutex
)

// SetGlobalLevel 设置全局默认日志级别
func SetGlobalLevel(level LogLevel) {
	globalLevelMu.Lock()
	defer globalLevelMu.Unlock()
	globalLevel = level
}

// GetGlobalLevel 获取全局默认日志级别
func GetGlobalLevel() LogLevel {
	globalLevelMu.RLock()
	defer globalLevelMu.RUnlock()
	return globalLevel
}

// Logger 日志记录器
type Logger struct {
	*log.Logger
	level LogLevel
	mu    sync.RWMutex
}

// NewLogger 创建新的日志记录器（使用全局级别）
func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", log.LstdFlags),
		level:  GetGlobalLevel(),
	}
}

// NewLoggerWithLevel 创建指定日志级别的日志记录器
func NewLoggerWithLevel(level LogLevel) *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", log.LstdFlags),
		level:  level,
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel 获取当前日志级别
func (l *Logger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// formatLog 格式化日志消息
func (l *Logger) formatLog(level LogLevel, message string) string {
	return fmt.Sprintf("[%s] %s", level.String(), message)
}

// shouldLog 判断是否应该输出该级别的日志
func (l *Logger) shouldLog(level LogLevel) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level >= l.level
}

// Debug 记录调试级别日志
func (l *Logger) Debug(format string, v ...interface{}) {
	if !l.shouldLog(DEBUG) {
		return
	}
	message := fmt.Sprintf(format, v...)
	l.Println(l.formatLog(DEBUG, message))
}

// Info 记录信息级别日志
func (l *Logger) Info(format string, v ...interface{}) {
	if !l.shouldLog(INFO) {
		return
	}
	message := fmt.Sprintf(format, v...)
	l.Println(l.formatLog(INFO, message))
}

// Warn 记录警告级别日志
func (l *Logger) Warn(format string, v ...interface{}) {
	if !l.shouldLog(WARN) {
		return
	}
	message := fmt.Sprintf(format, v...)
	l.Println(l.formatLog(WARN, message))
}

// Error 记录错误级别日志
func (l *Logger) Error(format string, v ...interface{}) {
	if !l.shouldLog(ERROR) {
		return
	}
	message := fmt.Sprintf(format, v...)
	l.Println(l.formatLog(ERROR, message))
}
