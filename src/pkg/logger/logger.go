package logger

import (
	"context"
	"os"
	"sync/atomic"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 编译期断言：*Logger 实现 ILogger
var _ ILogger = (*Logger)(nil)

type ILogger interface {
	Debugf(ctx context.Context, format string, v ...any)
	DebugW(ctx context.Context, msg string, keysAndValues ...any)
	Infof(ctx context.Context, format string, v ...any)
	InfoW(ctx context.Context, msg string, keysAndValues ...any)
	Warnf(ctx context.Context, format string, v ...any)
	WarnW(ctx context.Context, msg string, keysAndValues ...any)
	Errorf(ctx context.Context, format string, v ...any)
	ErrorW(ctx context.Context, msg string, keysAndValues ...any)
	Fatalf(ctx context.Context, format string, v ...any)
	FatalW(ctx context.Context, msg string, keysAndValues ...any)
	Panicf(ctx context.Context, format string, v ...any)
	PanicW(ctx context.Context, msg string, keysAndValues ...any)
}

type contextKey string

const RequestIDKey contextKey = "x_request_id"

type Logger struct {
	sugar *zap.SugaredLogger
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

func (l *Logger) withCtx(ctx context.Context) *zap.SugaredLogger {
	if rid, ok := ctx.Value(RequestIDKey).(string); ok && rid != "" {
		return l.sugar.With("request_id", rid)
	}
	return l.sugar
}

func (l *Logger) Debugf(ctx context.Context, format string, v ...any) {
	l.withCtx(ctx).Debugf(format, v...)
}
func (l *Logger) DebugW(ctx context.Context, msg string, keysAndValues ...any) {
	l.withCtx(ctx).Debugw(msg, keysAndValues...)
}
func (l *Logger) Infof(ctx context.Context, format string, v ...any) {
	l.withCtx(ctx).Infof(format, v...)
}
func (l *Logger) InfoW(ctx context.Context, msg string, keysAndValues ...any) {
	l.withCtx(ctx).Infow(msg, keysAndValues...)
}
func (l *Logger) Warnf(ctx context.Context, format string, v ...any) {
	l.withCtx(ctx).Warnf(format, v...)
}
func (l *Logger) WarnW(ctx context.Context, msg string, keysAndValues ...any) {
	l.withCtx(ctx).Warnw(msg, keysAndValues...)
}
func (l *Logger) Errorf(ctx context.Context, format string, v ...any) {
	l.withCtx(ctx).Errorf(format, v...)
}
func (l *Logger) ErrorW(ctx context.Context, msg string, keysAndValues ...any) {
	l.withCtx(ctx).Errorw(msg, keysAndValues...)
}
func (l *Logger) Fatalf(ctx context.Context, format string, v ...any) {
	l.withCtx(ctx).Fatalf(format, v...)
}
func (l *Logger) FatalW(ctx context.Context, msg string, keysAndValues ...any) {
	l.withCtx(ctx).Fatalw(msg, keysAndValues...)
}
func (l *Logger) Panicf(ctx context.Context, format string, v ...any) {
	l.withCtx(ctx).Panicf(format, v...)
}
func (l *Logger) PanicW(ctx context.Context, msg string, keysAndValues ...any) {
	l.withCtx(ctx).Panicw(msg, keysAndValues...)
}

// 包级默认实例，atomic.Value 存储 ILogger 接口，保证并发安全
var defaultLogger atomic.Value

func init() {
	defaultLogger.Store(newLoggerWithConfig(LogConfig{}))
}

func getDefault() ILogger {
	return defaultLogger.Load().(ILogger)
}

func newLoggerWithConfig(cfg LogConfig) ILogger {
	// 日志级别
	level := zapcore.InfoLevel
	if lvl, err := zapcore.ParseLevel(cfg.Level); err == nil {
		level = lvl
	}

	// 编码器
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "time"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encCfg)
	}

	// 写入目标
	var syncer zapcore.WriteSyncer
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
		if cfg.Console {
			syncer = zapcore.NewMultiWriteSyncer(file, zapcore.AddSync(os.Stdout))
		} else {
			syncer = file
		}
	} else {
		syncer = zapcore.AddSync(os.Stdout)
	}

	core := zapcore.NewCore(encoder, syncer, level)
	z := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	return &Logger{sugar: z.Sugar()}
}

func Infof(ctx context.Context, format string, v ...any) {
	getDefault().Infof(ctx, format, v...)
}

func Infow(ctx context.Context, msg string, keysAndValues ...interface{}) {
	getDefault().InfoW(ctx, msg, keysAndValues...)
}

func Errorf(ctx context.Context, format string, v ...any) {
	getDefault().Errorf(ctx, format, v...)
}

func Warnf(ctx context.Context, format string, v ...any) {
	getDefault().Warnf(ctx, format, v...)
}

func Debugf(ctx context.Context, format string, v ...any) {
	getDefault().Debugf(ctx, format, v...)
}
