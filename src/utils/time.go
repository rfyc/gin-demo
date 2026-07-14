package utils

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type runTime struct{}
type toolRunTime struct{}
type llmRunInfo struct {
	UUID      string    `json:"uuid"`
	LlmName   string    `json:"llm_name"`
	BeginTime time.Time `json:"begin_time"`
}

func CtxSetLlmRunInfo(ctx context.Context, llmName string) context.Context {
	return context.WithValue(ctx, "LLM_NAME", &llmRunInfo{
		UUID:      uuid.NewString(),
		LlmName:   llmName,
		BeginTime: time.Now(),
	})
}

func CtxGetLlmRunInfo(ctx context.Context) *llmRunInfo {
	if runInfo, ok := ctx.Value("LLM_NAME").(*llmRunInfo); ok && runInfo != nil {
		return runInfo
	}
	return nil
}

// CtxGetLlmRunName 获取当前 ctx 中 LLM 调用的名称
func CtxGetLlmRunName(ctx context.Context) string {
	if runInfo := CtxGetLlmRunInfo(ctx); runInfo != nil {
		return runInfo.LlmName
	}
	return ""
}

func CtxSetRunTime(ctx context.Context) context.Context {
	return context.WithValue(ctx, runTime{}, time.Now())
}

func CtxGetRunTime(ctx context.Context) time.Time {
	if beginTime, ok := ctx.Value(runTime{}).(time.Time); ok {
		return beginTime
	}
	return time.Time{}
}

// CtxSetToolRunTime 记录 Tool 开始时间(per-ctx隔离,不会被内部ChatModel覆盖)
func CtxSetToolRunTime(ctx context.Context) context.Context {
	return context.WithValue(ctx, toolRunTime{}, time.Now())
}

func CtxGetToolRunTime(ctx context.Context) time.Time {
	if beginTime, ok := ctx.Value(toolRunTime{}).(time.Time); ok {
		return beginTime
	}
	return time.Time{}
}

func CtxSetSinceSeconds(ctx context.Context, sinceSeconds float64) context.Context {
	return context.WithValue(ctx, "sinceSeconds", sinceSeconds)
}

func CtxGetSinceSeconds(ctx context.Context) (seconds float64) {
	if sinceSeconds, ok := ctx.Value("sinceSeconds").(float64); ok {
		return sinceSeconds
	}
	return 0
}

func TimeComparison(timestamp int64) string {
	inputTime := time.Unix(timestamp, 0)
	now := time.Now()
	currentDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	inputDate := time.Date(inputTime.Year(), inputTime.Month(), inputTime.Day(), 0, 0, 0, 0, inputTime.Location())

	switch inputDate.Sub(currentDate).Hours() {
	case 0:
		return "今天"
	case 24:
		return "明天"
	case 48:
		return "后天"
	default:
		return WeekDayToChinese(time.Unix(timestamp, 0).Weekday())
	}
}

// WeekDayToChinese 将time.Weekday类型转换为中文星期几
func WeekDayToChinese(weekday time.Weekday) string {
	weekdays := []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}
	return weekdays[weekday]
}

func GetLocalNowTime() time.Time {
	// 获取本地时区
	loc, _ := time.LoadLocation("Local")
	// 获取当前时间
	t := time.Now().In(loc)
	// 格式化输出
	return t
}

func GetDate() time.Time {
	date, _ := time.Parse("2006-01-02 15:04:05  -0700 MST", time.Now().Format("2006-01-02 15:04:05")+" +0800 CST")
	return date
}
func GetStringDate() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func TimeToDateTime(in time.Time) string {
	return in.Format("2006-01-02 15:04:05")
}
