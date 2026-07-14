package utils

import (
	"artist/application/pkg/logger"
	"context"
	"fmt"
	"os"
	"runtime"
	"time"
)

func Recover(ctx context.Context) {
	if err := recover(); err != nil {
		buf := make([]byte, 64<<10) //nolint:gomnd
		n := runtime.Stack(buf, false)
		buf = buf[:n]
		logger.Errorf(ctx, "recover panic: %v --- %s", err, string(buf))
		var currTime = time.Now().Format("2006-01-02 15:04:05.000")
		fmt.Fprintf(os.Stderr, "%s PANIC: %v \n%s\n", currTime, err, buf)
		fmt.Fprintf(os.Stdout, "%s PANIC: %v \n%s\n", currTime, err, buf)
	}
}
