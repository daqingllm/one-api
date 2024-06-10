package logger

import (
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	"time"
)

// LoggerWithRotation 包装 lumberjack.Logger 以支持定时分割
type LoggerWithRotation struct {
	Logger *lumberjack.Logger
}

// NewLoggerWithRotation 创建并初始化 LoggerWithRotation
func NewLoggerWithRotation(filename string, maxSize int) *LoggerWithRotation {
	l := &LoggerWithRotation{
		Logger: &lumberjack.Logger{
			Filename:   filename,
			MaxSize:    maxSize, // megabytes
			MaxBackups: 30,
			MaxAge:     7,    //days
			Compress:   true, // enabled by default
		},
	}
	go l.scheduleRotation()
	return l
}

// scheduleRotation 设置每天北京时间 02:00 触发日志分割
func (l *LoggerWithRotation) scheduleRotation() {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Fatalf("Failed to load location: %v", err)
	}
	for {
		now := time.Now().In(loc)
		// 计算下一个 02:00 的时间
		next := now.Add(time.Hour * 24)
		next = time.Date(next.Year(), next.Month(), next.Day(), 2, 0, 0, 0, loc)
		duration := next.Sub(now)
		time.AfterFunc(duration, func() {
			l.Logger.Rotate()
			l.scheduleRotation()
		})
		break
	}
}

// Write 实现 io.Writer 接口
func (l *LoggerWithRotation) Write(p []byte) (n int, err error) {
	return l.Logger.Write(p)
}
