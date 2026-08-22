package utils

import (
	"context"
	"fmt"
	"gin-demo/src/pkg/logger"
	"time"
)

func LogErrorInfo(ctx context.Context, bTime time.Time, tag string, err error, argKVs ...any) {

	var (
		needMsg string // needMsg 拼接后的 kv 键值对日志内容
	)

	// 1. 拼接 kv 键值对: 形如 [key:value] 的序列
	for k := 0; k+1 < len(argKVs); k += 2 {
		needMsg += fmt.Sprintf("[%v:%+v] ", argKVs[k], argKVs[k+1])
	}

	// 2. 按错误有无分级输出: err 非空记 Error 并携带错误信息, 否则记 Info
	if err == nil {
		logger.Infof(ctx, "%s [since:%s] %s", tag, time.Since(bTime).String(), needMsg)
	} else {
		logger.Errorf(ctx, "%s [error:%v] [since:%s] %s", tag, err, time.Since(bTime).String(), needMsg)
	}
}
