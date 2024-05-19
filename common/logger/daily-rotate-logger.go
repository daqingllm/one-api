package logger

import (
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	"os"
	"path/filepath"
	"time"
)

// DailyRotateWriter 实现每天北京时间凌晨2:00分割日志的 io.Writer
type DailyRotateWriter struct {
	*lumberjack.Logger
	lastRotate time.Time
}

// NewDailyRotateWriter 创建一个新的 DailyRotateWriter
func NewDailyRotateWriter(filename string, maxSize int) *DailyRotateWriter {
	dir := filepath.Dir(filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create log directory: %v", err)
		}
	}

	w := &DailyRotateWriter{
		Logger: &lumberjack.Logger{
			Filename:   filename,
			MaxSize:    maxSize, // 单个日志文件最大 10MB
			MaxBackups: 30,
			MaxAge:     7, // 保留 7 天内的日志文件
			Compress:   true,
		},
		lastRotate: time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 2, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*3600)),
	}

	// 每天北京时间凌晨2:00进行日志文件分割
	go w.runDailyRotate()

	return w
}

func (w *DailyRotateWriter) Write(p []byte) (n int, err error) {
	// 检查是否需要进行日志文件分割
	if time.Now().In(time.FixedZone("Asia/Shanghai", 8*3600)).After(w.lastRotate.Add(24 * time.Hour)) {
		w.Rotate()
		w.lastRotate = w.lastRotate.Add(24 * time.Hour)
	}

	return w.Logger.Write(p)
}

func (w *DailyRotateWriter) runDailyRotate() {
	for {
		// 等待到下一个北京时间凌晨2:00
		nextRotate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day()+1, 2, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*3600))
		time.Sleep(nextRotate.Sub(time.Now().In(time.FixedZone("Asia/Shanghai", 8*3600))))

		// 进行日志文件分割
		w.Rotate()
		w.lastRotate = nextRotate
	}
}
