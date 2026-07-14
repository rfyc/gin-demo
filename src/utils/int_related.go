package utils

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"strconv"
	"strings"
)

// GetQueryInt64 获取int64参数
func GetQueryInt64(ctx *gin.Context, param string) (int64, error) {
	paramStr := ctx.Query(param)
	return strconv.ParseInt(paramStr, 10, 64)
}

// GetQueryInt 获取int参数
func GetQueryInt(ctx *gin.Context, param string) (int, error) {
	paramStr := ctx.Query(param)
	return strconv.Atoi(paramStr)
}

func GetInt64Join(list []int64, sep string) (str string) {
	for _, item := range list {
		str += strconv.FormatInt(item, 10) + sep
	}
	str = strings.TrimRight(str, sep)
	return
}

func GetIntJoin(list []int, sep string) (str string) {
	for _, item := range list {
		str += cast.ToString(item) + sep
	}
	str = strings.TrimRight(str, sep)
	return
}
