package redislock

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// 获取当前进程的 pid
func GetCurrentPid() string {
	return strconv.Itoa(os.Getpid())
}

func GetCurrentGoroutineID() string {
	buf := make([]byte, 128)
	buf = buf[:runtime.Stack(buf, false)]
	stackInfo := string(buf)
	return strings.TrimSpace(strings.Split(stackInfo, "goroutine ")[1])
}
func GetProcessAndGoroutineIDStr() string {
	return fmt.Sprintf("%s_%s", GetCurrentPid(), GetCurrentGoroutineID())
}
