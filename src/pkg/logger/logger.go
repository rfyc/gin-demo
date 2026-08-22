// Package logger 提供基于 zap 的日志封装:
// 同时输出到控制台与文件, 支持按级别过滤、按天滚动切割,
// 并可自动从 context 中读取 request_id 注入日志字段。
package logger

import (
	"context"
	"os"
	"sync/atomic"
	"time"

	"gin-demo/src/schema"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 编译期断言: 确保 Logger 实现 ILogger 接口。
var _ ILogger = (*Logger)(nil)

// ILogger 是日志对外接口, 区分带 context 的业务方法与包级 Print 系列方法。
type ILogger interface {
	Infof(ctx context.Context, format string, v ...any)
	Info(ctx context.Context, msg string, keysAndValues ...any)
	Warnf(ctx context.Context, format string, v ...any)
	Warn(ctx context.Context, msg string, keysAndValues ...any)
	Errorf(ctx context.Context, format string, v ...any)
	Error(ctx context.Context, msg string, keysAndValues ...any)
	Printf(format string, v ...any)
	Print(msg string, keysAndValues ...any)
}

// Logger 是 ILogger 的具体实现, 持有多种 zap 日志实例:
//   - sugar/sugarPkg: 带调用层级偏移的日志, 供方法与包级函数分别使用
//   - printSugar/printSugarPkg: 强制输出到控制台的日志, 供 Print 系列使用
type Logger struct {
	sugar         *zap.SugaredLogger // sugar 方法级日志实例
	sugarPkg      *zap.SugaredLogger // sugarPkg 包级日志实例
	printSugar    *zap.SugaredLogger // printSugar 方法级强制控制台日志实例
	printSugarPkg *zap.SugaredLogger // printSugarPkg 包级强制控制台日志实例
}

// LogConfig 是日志配置, 与配置文件 Log 节点对应。
type LogConfig struct {
	Path       string `mapstructure:"path"`       // Path 日志文件路径, 为空时仅输出控制台
	Level      string `mapstructure:"level"`      // Level 日志级别, 如 debug/info/warn/error
	Format     string `mapstructure:"format"`     // Format 日志格式(预留)
	MaxSize    int    `mapstructure:"maxSize"`    // MaxSize 单个日志文件最大体积(MB)
	MaxAge     int    `mapstructure:"maxAge"`     // MaxAge 日志文件保留天数
	MaxBackups int    `mapstructure:"maxBackups"` // MaxBackups 保留的旧日志文件数
	Compress   bool   `mapstructure:"compress"`   // Compress 是否压缩历史日志
	Console    bool   `mapstructure:"console"`    // Console 是否同时输出到控制台
}

// NewLogger 基于配置创建日志实例, 并设为包级默认日志。
//
// 参数:
//   - cfg: 日志配置
//
// 返回:
//   - ILogger: 已初始化的日志实例
func NewLogger(cfg LogConfig) ILogger {
	var l ILogger // 创建的日志实例

	l = newLoggerWithConfig(cfg)
	defaultLogger.Store(l)
	return l
}

// withCtx 根据 ctx 中的 request_id 为日志附加字段。
//
// 参数:
//   - ctx: 请求上下文, 含 request_id 时附加该字段
//   - pkg: 是否为包级函数调用
//
// 返回:
//   - *zap.SugaredLogger: 附加 request_id 后的日志实例
func (l *Logger) withCtx(ctx context.Context, pkg bool) *zap.SugaredLogger {

	var s *zap.SugaredLogger // 基础日志实例, 按 pkg 选择方法级或包级
	s = l.sugar
	if pkg {
		s = l.sugarPkg
	}
	if rid, ok := ctx.Value(schema.CTX_TraceIDKey).(string); ok && rid != "" {
		return s.With("request_id", rid)
	}
	return s
}

// rawSugar 按 pkg 与 printMode 组合返回对应的日志实例。
//
// 参数:
//   - pkg: 是否为包级函数调用
//   - printMode: 是否使用强制控制台输出的实例
//
// 返回:
//   - *zap.SugaredLogger: 匹配的日志实例
func (l *Logger) rawSugar(pkg bool, printMode bool) *zap.SugaredLogger {

	switch {
	case pkg && printMode:
		return l.printSugarPkg
	case pkg:
		return l.sugarPkg
	case printMode:
		return l.printSugar
	default:
		return l.sugar
	}
}

// Infof 记录 info 级别格式化日志, 自动附加 ctx 中的 request_id。
func (l *Logger) Infof(ctx context.Context, format string, v ...any) {

	l.withCtx(ctx, false).Infof(format, v...)
}

// Info 记录 info 级别键值对日志, 自动附加 ctx 中的 request_id。
func (l *Logger) Info(ctx context.Context, msg string, keysAndValues ...any) {

	l.withCtx(ctx, false).Infow(msg, keysAndValues...)
}

// Warnf 记录 warn 级别格式化日志, 自动附加 ctx 中的 request_id。
func (l *Logger) Warnf(ctx context.Context, format string, v ...any) {

	l.withCtx(ctx, false).Warnf(format, v...)
}

// Warn 记录 warn 级别键值对日志, 自动附加 ctx 中的 request_id。
func (l *Logger) Warn(ctx context.Context, msg string, keysAndValues ...any) {

	l.withCtx(ctx, false).Warnw(msg, keysAndValues...)
}

// Errorf 记录 error 级别格式化日志, 自动附加 ctx 中的 request_id。
func (l *Logger) Errorf(ctx context.Context, format string, v ...any) {

	l.withCtx(ctx, false).Errorf(format, v...)
}

// Error 记录 error 级别键值对日志, 自动附加 ctx 中的 request_id。
func (l *Logger) Error(ctx context.Context, msg string, keysAndValues ...any) {

	l.withCtx(ctx, false).Errorw(msg, keysAndValues...)
}

// Printf 以 info 级别输出格式化日志, 并强制同步刷新到控制台。
func (l *Logger) Printf(format string, v ...any) {

	l.rawSugar(false, true).Infof(format, v...)
	_ = l.printSugar.Sync()
}

// Print 以 info 级别输出键值对日志, 并强制同步刷新到控制台。
func (l *Logger) Print(msg string, keysAndValues ...any) {

	l.rawSugar(false, true).Infow(msg, keysAndValues...)
	_ = l.printSugar.Sync()
}

// defaultLogger 保存包级默认日志实例, 通过原子值保证并发安全。
var defaultLogger atomic.Value

// init 在包导入时以空配置创建默认日志实例, 保证未显式 NewLogger 时也可用。
func init() {

	defaultLogger.Store(newLoggerWithConfig(LogConfig{}))
}

// getDefault 返回包级默认日志实例。
func getDefault() ILogger {

	return defaultLogger.Load().(ILogger)
}

// getDefaultLogger 返回包级默认 *Logger 实例; 类型不匹配时返回 nil。
func getDefaultLogger() *Logger {

	if l, ok := getDefault().(*Logger); ok {
		return l
	}
	return nil
}

// buildDualCores 构建双输出日志核心: 控制台(console)与文件(JSON), 通过 zapcore.NewTee 组合。
//
// 参数:
//   - cfg: 日志配置, 决定级别、文件路径与保留策略
//   - forceStdout: 为 true 时强制输出到控制台
//
// 返回:
//   - zapcore.Core: 组合后的日志核心
func buildDualCores(cfg LogConfig, forceStdout bool) zapcore.Core {
	var (
		levelEnabler  zapcore.LevelEnabler         // levelEnabler 日志级别过滤器
		baseEncCfg    zapcore.EncoderConfig        // baseEncCfg 基础编码配置
		fileEncCfg    zapcore.EncoderConfig        // fileEncCfg 文件(JSON)编码配置
		consoleEncCfg zapcore.EncoderConfig        // consoleEncCfg 控制台编码配置
		stdout        = zapcore.AddSync(os.Stdout) // stdout 标准输出同步器
		cores         []zapcore.Core               // cores 日志核心列表
	)

	// 1. 构建日志级别过滤器
	{
		levelEnabler = zap.LevelEnablerFunc(func(l zapcore.Level) bool {

			var level = zapcore.InfoLevel // level 生效的日志级别下限
			if lvl, err := zapcore.ParseLevel(cfg.Level); err == nil {
				level = lvl
			}
			return l >= level
		})
	}

	// 2. 组装编码配置: 时间格式与级别颜色
	{
		const reset = "\x1b[0m" // reset 终端颜色重置转义码

		baseEncCfg = zap.NewProductionEncoderConfig()
		baseEncCfg.TimeKey = "time"
		baseEncCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")

		fileEncCfg = baseEncCfg
		fileEncCfg.EncodeLevel = zapcore.CapitalLevelEncoder

		consoleEncCfg = baseEncCfg
		consoleEncCfg.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
		}
		consoleEncCfg.EncodeCaller = zapcore.ShortCallerEncoder
		consoleEncCfg.EncodeLevel = func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			var (
				inner      = level.CapitalString() // inner 级别的全大写字符串
				levelColor string                  // levelColor 级别对应的终端颜色码
			)
			switch level {
			case zapcore.WarnLevel:
				levelColor = "\x1b[33;1m"
			case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
				levelColor = "\x1b[31;1m"
			case zapcore.DebugLevel:
				levelColor = "\x1b[36m"
			default:
				levelColor = "\x1b[32;1m"
			}
			enc.AppendString(levelColor + inner + reset)
		}
	}

	// 3. 配置要求输出控制台时, 追加控制台核心
	{
		var (
			consoleEnc zapcore.Encoder // consoleEnc 控制台编码器
		)
		if forceStdout || cfg.Console || cfg.Path == "" {
			consoleEnc = zapcore.NewConsoleEncoder(consoleEncCfg)
			cores = append(cores, zapcore.NewCore(consoleEnc, stdout, levelEnabler))
		}
	}

	// 4. 配置了文件路径时, 追加文件(JSON)核心, 并按配置滚动切割
	{
		var (
			maxSize    int                 // maxSize 单文件体积上限(MB), 默认 100
			maxAge     int                 // maxAge 历史文件保留天数, 默认 30
			maxBackups int                 // maxBackups 保留的旧文件数, 默认 10
			file       zapcore.WriteSyncer // file 滚动切割的文件同步器
			jsonEnc    zapcore.Encoder     // jsonEnc 文件(JSON)编码器
		)
		if cfg.Path != "" {
			maxSize = cfg.MaxSize
			if maxSize <= 0 {
				maxSize = 100
			}
			maxAge = cfg.MaxAge
			if maxAge <= 0 {
				maxAge = 30
			}
			maxBackups = cfg.MaxBackups
			if maxBackups <= 0 {
				maxBackups = 10
			}
			file = zapcore.AddSync(&lumberjack.Logger{
				Filename:   cfg.Path,
				MaxSize:    maxSize,
				MaxAge:     maxAge,
				MaxBackups: maxBackups,
				Compress:   cfg.Compress,
			})
			jsonEnc = zapcore.NewJSONEncoder(fileEncCfg)
			cores = append(cores, zapcore.NewCore(jsonEnc, file, levelEnabler))
		}
	}

	// 5. 兜底: 至少一个核心, 组合多路输出
	if len(cores) == 0 {
		cores = append(cores, zapcore.NewCore(zapcore.NewConsoleEncoder(consoleEncCfg), stdout, levelEnabler))
	}
	return zapcore.NewTee(cores...)
}

// newLoggerWithConfig 基于配置创建 Logger 实例。
// 使用 AddCallerSkip(1) 使日志中的文件行号指向实际调用方。
func newLoggerWithConfig(cfg LogConfig) ILogger {
	var (
		core      zapcore.Core // 常规日志核心(文件+控制台)
		printCore zapcore.Core // 强制控制台日志核心
		zSkip1    *zap.Logger  // 常规日志实例
		zPrint    *zap.Logger  // 强制控制台日志实例
	)

	core = buildDualCores(cfg, false)
	printCore = buildDualCores(cfg, true)

	zSkip1 = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	zPrint = zap.New(printCore, zap.AddCaller(), zap.AddCallerSkip(1))
	return &Logger{
		sugar:         zSkip1.Sugar(),
		sugarPkg:      zSkip1.Sugar(),
		printSugar:    zPrint.Sugar(),
		printSugarPkg: zPrint.Sugar(),
	}
}

// Infof 记录 info 级别格式化日志, 自动附加 ctx 中的 request_id。
func Infof(ctx context.Context, format string, v ...any) {

	if l := getDefaultLogger(); l != nil {
		l.withCtx(ctx, true).Infof(format, v...)
	}
}

// Info 记录 info 级别键值对日志, 自动附加 ctx 中的 request_id。
func Info(ctx context.Context, msg string, keysAndValues ...interface{}) {

	if l := getDefaultLogger(); l != nil {
		l.withCtx(ctx, true).Infow(msg, keysAndValues...)
	}
}

// Warnf 记录 warn 级别格式化日志, 自动附加 ctx 中的 request_id。
func Warnf(ctx context.Context, format string, v ...any) {

	if l := getDefaultLogger(); l != nil {
		l.withCtx(ctx, true).Warnf(format, v...)
	}
}

// Warn 记录 warn 级别键值对日志, 自动附加 ctx 中的 request_id。
func Warn(ctx context.Context, msg string, keysAndValues ...interface{}) {

	if l := getDefaultLogger(); l != nil {
		l.withCtx(ctx, true).Warnw(msg, keysAndValues...)
	}
}

// Errorf 记录 error 级别格式化日志, 自动附加 ctx 中的 request_id。
func Errorf(ctx context.Context, format string, v ...any) {

	if l := getDefaultLogger(); l != nil {
		l.withCtx(ctx, true).Errorf(format, v...)
	}
}

// Error 记录 error 级别键值对日志, 自动附加 ctx 中的 request_id。
func Error(ctx context.Context, msg string, keysAndValues ...interface{}) {

	if l := getDefaultLogger(); l != nil {
		l.withCtx(ctx, true).Errorw(msg, keysAndValues...)
	}
}

// Printf 以 info 级别输出格式化日志, 并强制同步刷新到控制台。
func Printf(format string, v ...any) {

	if l := getDefaultLogger(); l != nil {
		l.rawSugar(true, true).Infof(format, v...)
		_ = l.printSugar.Sync()
	}
}

// Print 以 info 级别输出键值对日志, 并强制同步刷新到控制台。
func Print(msg string, keysAndValues ...interface{}) {

	if l := getDefaultLogger(); l != nil {
		l.rawSugar(true, true).Infow(msg, keysAndValues...)
		_ = l.printSugar.Sync()
	}
}

// Sync 刷新包级默认日志实例的缓冲, 供进程退出前调用。
func Sync() error {

	if l := getDefaultLogger(); l != nil && l.sugar != nil {
		l.sugar.Sync()
		if l.printSugar != nil {
			_ = l.printSugar.Sync()
		}
		return nil
	}
	return nil
}
