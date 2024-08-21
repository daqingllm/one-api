package helper

import (
	"fmt"
	"time"
)

func GetTimestamp() int64 {
	return time.Now().Unix()
}

func GetTimeString() string {
	now := time.Now()
	return fmt.Sprintf("%s%d", now.Format("20060102150405"), now.UnixNano()%1e9)
}

// GetFormattedTimeString 返回当前时间的格式化字符串，格式为 YYYY-MM-DD HH:MM:SS
func GetFormattedTimeString() string {
	now := time.Now()
	return now.Format("2006-01-02 15:04:05")
}
