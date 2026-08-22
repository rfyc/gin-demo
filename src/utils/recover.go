package utils

import (
	"context"
	"fmt"
	"gin-demo/src/pkg/logger"
	"os"
	"runtime"
	"time"
)

// Recover 捕获当前 goroutine 的 panic, 记录错误日志并输出到标准错误与标准输出。
//
// 参数:
//   - ctx: 请求上下文, 用于携带 request_id 等链路信息
func Recover(ctx context.Context) {

	if err := recover(); err != nil {
		var (
			currTime = time.Now().Format("2006-01-02 15:04:05.000") // currTime 当前时间, 格式 yyyy-MM-dd HH:mm:ss.SSS
			buf      []byte                                         // buf panic 堆栈信息缓冲
			n        int                                            // n 实际写入缓冲的堆栈字节数
		)

		buf = make([]byte, 64<<10) //nolint:gomnd
		n = runtime.Stack(buf, false)
		buf = buf[:n]

		logger.Errorf(ctx, "recover panic: %v --- %s", err, string(buf))
		fmt.Fprintf(os.Stderr, "%s PANIC: %v \n%s\n", currTime, err, buf)
		fmt.Fprintf(os.Stdout, "%s PANIC: %v \n%s\n", currTime, err, buf)
	}
}
