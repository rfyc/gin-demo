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
	sugar         *zap.SugaredLogger
	sugarPkg      *zap.SugaredLogger
	printSugar    *zap.SugaredLogger
	printSugarPkg *zap.SugaredLogger
}

type LogConfig struct {
	Path       string `mapstructure:"path"`
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	MaxSize    int    `mapstructure:"maxSize"`
	MaxAge     int    `mapstructure:"maxAge"`
	MaxBackups int    `mapstructure:"maxBackups"`
	Compress   bool   `mapstructure:"compress"`
	Console    bool   `mapstructure:"console"`
}

func NewLogger(cfg LogConfig) ILogger {
	l := newLoggerWithConfig(cfg)
	defaultLogger.Store(l)
	return l
}

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

var defaultLogger atomic.Value

func init() {
	defaultLogger.Store(newLoggerWithConfig(LogConfig{}))
}

func getDefault() ILogger {
	return defaultLogger.Load().(ILogger)
}

func getDefaultLogger() *Logger {
	if l, ok := getDefault().(*Logger); ok {
		return l
	}
	return nil
}

func buildDualCores(cfg LogConfig, forceStdout bool) zapcore.Core {
	levelEnabler := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		level := zapcore.InfoLevel
		if lvl, err := zapcore.ParseLevel(cfg.Level); err == nil {
			level = lvl
		}
		return l >= level
	})

	baseEncCfg := zap.NewProductionEncoderConfig()
	baseEncCfg.TimeKey = "time"
	baseEncCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")

	fileEncCfg := baseEncCfg
	fileEncCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	const reset = "\x1b[0m"
	consoleEncCfg := baseEncCfg
	consoleEncCfg.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}
	consoleEncCfg.EncodeCaller = zapcore.ShortCallerEncoder
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

	if forceStdout || cfg.Console || cfg.Path == "" {
		consoleEnc := zapcore.NewConsoleEncoder(consoleEncCfg)
		cores = append(cores, zapcore.NewCore(consoleEnc, stdout, levelEnabler))
	}

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
		cores = append(cores, zapcore.NewCore(zapcore.NewConsoleEncoder(consoleEncCfg), stdout, levelEnabler))
	}
	return zapcore.NewTee(cores...)
}

func newLoggerWithConfig(cfg LogConfig) ILogger {
	core := buildDualCores(cfg, false)
	printCore := buildDualCores(cfg, true)

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
