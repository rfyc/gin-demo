package logger

import (
	"context"
	"os"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 编译期断言：*Logger 实现 ILogger
var _ ILogger = (*Logger)(nil)

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

type contextKey string

const RequestIDKey contextKey = "x_request_id"

type Logger struct {
	// sugar 给 Logger 结构体方法调用用，跳过 1 层（方法本身）
	sugar *zap.SugaredLogger
	// sugarPkg 给包级便捷函数调用用，跳过 1 层（与方法调用相同，因内部直接调 withCtx）
	sugarPkg *zap.SugaredLogger
	// printSugar 给 Printf/Print 专用，强制输出到 stdout（不受 Console 配置限制），
	// 如配置了 Path 也同步写文件，确保调试输出既能看到又能被日志归档。
	printSugar *zap.SugaredLogger
	// printSugarPkg 是包级 Printf/Print 专用，调用栈与 sugar/sugarPkg 对齐。
	printSugarPkg *zap.SugaredLogger
}

// LogConfig 控制 zap logger 的行为
type LogConfig struct {
	Path       string `mapstructure:"path"`       // 日志文件路径；为空时只输出到 stdout
	Level      string `mapstructure:"level"`      // 日志级别：debug / info / warn / error，默认 info
	Format     string `mapstructure:"format"`     // 输出格式：json / console，默认 json
	MaxSize    int    `mapstructure:"maxSize"`    // 单个文件最大 MB，默认 100
	MaxAge     int    `mapstructure:"maxAge"`     // 文件保留天数，默认 30
	MaxBackups int    `mapstructure:"maxBackups"` // 最大备份文件数，默认 10
	Compress   bool   `mapstructure:"compress"`   // 是否 gzip 压缩归档文件
	Console    bool   `mapstructure:"console"`    // Path 非空时是否同时输出到 stdout
}

func NewLogger(cfg LogConfig) ILogger {
	l := newLoggerWithConfig(cfg)
	defaultLogger.Store(l)
	return l
}

// withCtx 根据使用场景（方法调用 / 包级调用）返回附加了 request_id 的 sugared logger。
// pkg=true 时返回 sugarPkg（跳过更多层），否则返回 sugar（给结构体方法用）。
func (l *Logger) withCtx(ctx context.Context, pkg bool) *zap.SugaredLogger {
	s := l.sugar
	if pkg {
		s = l.sugarPkg
	}
	if rid, ok := ctx.Value(RequestIDKey).(string); ok && rid != "" {
		return s.With("request_id", rid)
	}
	return s
}

// rawSugar 返回不带 request_id 的 sugared logger，用于 Printf/Print。
// pkg=true 时使用包级调用专用的 sugarPkg / printSugarPkg。
// printMode=true 时使用 Printf/Print 专用的强制 stdout sugar。
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

func (l *Logger) Infof(ctx context.Context, format string, v ...any) {
	l.withCtx(ctx, false).Infof(format, v...)
}
func (l *Logger) Info(ctx context.Context, msg string, keysAndValues ...any) {
	l.withCtx(ctx, false).Infow(msg, keysAndValues...)
}

func (l *Logger) Warnf(ctx context.Context, format string, v ...any) {
	l.withCtx(ctx, false).Warnf(format, v...)
}
func (l *Logger) Warn(ctx context.Context, msg string, keysAndValues ...any) {
	l.withCtx(ctx, false).Warnw(msg, keysAndValues...)
}

func (l *Logger) Errorf(ctx context.Context, format string, v ...any) {
	l.withCtx(ctx, false).Errorf(format, v...)
}
func (l *Logger) Error(ctx context.Context, msg string, keysAndValues ...any) {
	l.withCtx(ctx, false).Errorw(msg, keysAndValues...)
}

func (l *Logger) Printf(format string, v ...any) {
	l.rawSugar(false, true).Infof(format, v...)
	_ = l.printSugar.Sync()
}
func (l *Logger) Print(msg string, keysAndValues ...any) {
	l.rawSugar(false, true).Infow(msg, keysAndValues...)
	_ = l.printSugar.Sync()
}

// 包级默认实例，atomic.Value 存储 ILogger 接口，保证并发安全
var defaultLogger atomic.Value

func init() {
	defaultLogger.Store(newLoggerWithConfig(LogConfig{}))
}

func getDefault() ILogger {
	return defaultLogger.Load().(ILogger)
}

// getDefaultLogger 以 *Logger 形式返回默认实例，便于包级函数使用 sugarPkg。
func getDefaultLogger() *Logger {
	if l, ok := getDefault().(*Logger); ok {
		return l
	}
	return nil
}

// buildDualCores 根据 cfg 构建两条独立的输出链路，使用 zapcore.NewTee 合并：
//
//   - 控制台输出 (stdout)：强制 ConsoleEncoder（人类友好格式），
//     当 cfg.Console=true 或 forceStdout=true 时启用（普通日志受 Console 开关控制，
//     Printf/Print 专用 forceStdout=true 保证始终打印终端）。
//
//   - 文件输出 (lumberjack)：强制 JSONEncoder（结构化格式），
//     仅当 cfg.Path 非空时启用。
//
// 返回：合并后的 tee core（单条日志同时去往已启用的所有目标）。
func buildDualCores(cfg LogConfig, forceStdout bool) zapcore.Core {
	// 日志级别（两个 core 共享）
	levelEnabler := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		level := zapcore.InfoLevel
		if lvl, err := zapcore.ParseLevel(cfg.Level); err == nil {
			level = lvl
		}
		return l >= level
	})

	// 通用 encoder 基础配置
	baseEncCfg := zap.NewProductionEncoderConfig()
	baseEncCfg.TimeKey = "time"
	baseEncCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")

	// 文件使用的 encoder 配置：级别用大写纯文本，避免 ANSI 颜色字符污染 JSON
	fileEncCfg := baseEncCfg
	fileEncCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	// 控制台使用的 encoder 配置：还原 zap 原始 console 字段展示风格（去掉方括号徽章），
	// 仅保留：自定义时间格式 + INFO/WARN/ERROR 等彩色级别。
	const reset = "\x1b[0m"

	consoleEncCfg := baseEncCfg
	// 不修改 ConsoleSeparator，保持 zap 原始默认 tab 分隔

	// 时间：原始纯文本格式 2006-01-02 15:04:05.000（无任何方括号）
	consoleEncCfg.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		const layout = "2006-01-02 15:04:05.000"
		enc.AppendString(t.Format(layout))
	}
	// caller：原始 TrimmedPath，不补空格、不加方括号
	consoleEncCfg.EncodeCaller = zapcore.ShortCallerEncoder
	// level：去掉方括号和补空格，直接大写彩色（对应 INFO / WARN / ERROR / DEBUG 原始风格）
	consoleEncCfg.EncodeLevel = func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		inner := level.CapitalString()
		var levelColor string
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

	var cores []zapcore.Core
	stdout := zapcore.AddSync(os.Stdout)

	// 1) 控制台 core：强制 Console 格式 + 彩色级别，使用原始 zap ConsoleEncoder 分隔方式
	if forceStdout || cfg.Console || cfg.Path == "" {
		consoleEnc := zapcore.NewConsoleEncoder(consoleEncCfg)
		cores = append(cores, zapcore.NewCore(consoleEnc, stdout, levelEnabler))
	}

	// 2) 文件 core：强制 JSON 格式 + 纯文本级别（无颜色转义）
	if cfg.Path != "" {
		maxSize := cfg.MaxSize
		if maxSize <= 0 {
			maxSize = 100
		}
		maxAge := cfg.MaxAge
		if maxAge <= 0 {
			maxAge = 30
		}
		maxBackups := cfg.MaxBackups
		if maxBackups <= 0 {
			maxBackups = 10
		}
		file := zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.Path,
			MaxSize:    maxSize,
			MaxAge:     maxAge,
			MaxBackups: maxBackups,
			Compress:   cfg.Compress,
		})
		jsonEnc := zapcore.NewJSONEncoder(fileEncCfg)
		cores = append(cores, zapcore.NewCore(jsonEnc, file, levelEnabler))
	}

	if len(cores) == 0 {
		// 兜底：至少有 stdout console
		cores = append(cores, zapcore.NewCore(zapcore.NewConsoleEncoder(consoleEncCfg), stdout, levelEnabler))
	}
	return zapcore.NewTee(cores...)
}

func newLoggerWithConfig(cfg LogConfig) ILogger {
	// 普通日志 core：受 Console 开关控制
	core := buildDualCores(cfg, false)
	// Printf/Print 专用 core：强制开启 stdout 输出（forceStdout=true）
	printCore := buildDualCores(cfg, true)

	// 说明：经过严格调用栈测试，无论是方法调用还是包级便捷函数（直接操作 *Logger 内部）
	// 都只包装了一层函数，因此统一使用 AddCallerSkip(1) 即可正确指向真正的调用方代码行。
	zSkip1 := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	zPrint := zap.New(printCore, zap.AddCaller(), zap.AddCallerSkip(1))
	return &Logger{
		sugar:         zSkip1.Sugar(),
		sugarPkg:      zSkip1.Sugar(),
		printSugar:    zPrint.Sugar(),
		printSugarPkg: zPrint.Sugar(),
	}
}

func Infof(ctx context.Context, format string, v ...any) {
	if l := getDefaultLogger(); l != nil {
		l.withCtx(ctx, true).Infof(format, v...)
	}
}

func Info(ctx context.Context, msg string, keysAndValues ...interface{}) {
	if l := getDefaultLogger(); l != nil {
		l.withCtx(ctx, true).Infow(msg, keysAndValues...)
	}
}

func Warnf(ctx context.Context, format string, v ...any) {
	if l := getDefaultLogger(); l != nil {
		l.withCtx(ctx, true).Warnf(format, v...)
	}
}

func Warn(ctx context.Context, msg string, keysAndValues ...interface{}) {
	if l := getDefaultLogger(); l != nil {
		l.withCtx(ctx, true).Warnw(msg, keysAndValues...)
	}
}

func Errorf(ctx context.Context, format string, v ...any) {
	if l := getDefaultLogger(); l != nil {
		l.withCtx(ctx, true).Errorf(format, v...)
	}
}

func Error(ctx context.Context, msg string, keysAndValues ...interface{}) {
	if l := getDefaultLogger(); l != nil {
		l.withCtx(ctx, true).Errorw(msg, keysAndValues...)
	}
}

func Printf(format string, v ...any) {
	if l := getDefaultLogger(); l != nil {
		l.rawSugar(true, true).Infof(format, v...)
		_ = l.printSugar.Sync()
	}
}

func Print(msg string, keysAndValues ...interface{}) {
	if l := getDefaultLogger(); l != nil {
		l.rawSugar(true, true).Infow(msg, keysAndValues...)
		_ = l.printSugar.Sync()
	}
}

// Sync 刷新底层缓冲到输出目标（文件或 stdout）。
// 默认 logger 使用的 zap core 是带缓冲的，短生命周期程序在退出前调用 Sync 可避免丢日志。
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
