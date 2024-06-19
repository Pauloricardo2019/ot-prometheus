package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Duration int64

const (
	Nanosecond  Duration = 1
	Microsecond          = 1000 * Nanosecond
	Millisecond          = 1000 * Microsecond
	Second               = 1000 * Millisecond
	Minute               = 60 * Second
	Hour                 = 60 * Minute
)

type LoggerConfig struct {
	Development       bool
	DisableCaller     bool
	DisableStacktrace bool
	Encoding          string
	Level             string
}

// Logger methods interface
type Logger interface {
	InitLogger(level string)
	Debug(args ...interface{})
	Debugf(template string, args ...interface{})
	Info(args ...interface{})
	Infof(template string, args ...interface{})
	Warn(args ...interface{})
	Warnf(template string, args ...interface{})
	Error(args ...interface{})
	Errorf(template string, args ...interface{})
	DPanic(args ...interface{})
	DPanicf(template string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(template string, args ...interface{})
	Chronometer(mensagem string, inicio *time.Time)
}

type ApiLogger struct {
	sugarLogger *zap.SugaredLogger
}

func NewApiLogger() *ApiLogger {
	return &ApiLogger{}
}

var loggerLevelMap = map[string]zapcore.Level{
	"debug":  zapcore.DebugLevel,
	"info":   zapcore.InfoLevel,
	"warn":   zapcore.WarnLevel,
	"error":  zapcore.ErrorLevel,
	"dpanic": zapcore.DPanicLevel,
	"panic":  zapcore.PanicLevel,
	"fatal":  zapcore.FatalLevel,
}

func (l *ApiLogger) getLoggerLevel(nivel_de_log string) zapcore.Level {
	level, exist := loggerLevelMap[nivel_de_log]
	if !exist {
		return zapcore.DebugLevel
	}

	return level
}

func (l *ApiLogger) InitLogger(level string) {

	logLevel := l.getLoggerLevel(level)

	logWriter := zapcore.AddSync(os.Stderr)

	var encoderCfg zapcore.EncoderConfig
	if "l.cfg.Server.Mode" == "Development" {
		encoderCfg = zap.NewDevelopmentEncoderConfig()
	} else {
		encoderCfg = zap.NewProductionEncoderConfig()
	}

	var encoder zapcore.Encoder
	encoderCfg.LevelKey = "LEVEL"
	encoderCfg.CallerKey = "CALLER"
	encoderCfg.TimeKey = "TIME"
	encoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout("Jan 02 15:04:05.000000000")
	encoderCfg.StacktraceKey = ""
	encoderCfg.NameKey = "NAME"
	encoderCfg.MessageKey = "MESSAGE"

	if "l.cfg.Logger.Encoding" == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(encoder, logWriter, zap.NewAtomicLevelAt(logLevel))
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	l.sugarLogger = logger.Sugar()
	if err := l.sugarLogger.Sync(); err != nil {
		l.sugarLogger.Error(err)
	}
}

// Logger methods

func (l *ApiLogger) Debug(args ...interface{}) {
	l.sugarLogger.Debug(args...)
}

func (l *ApiLogger) Debugf(template string, args ...interface{}) {
	l.sugarLogger.Debugf(template, args...)
}

func (l *ApiLogger) Info(args ...interface{}) {
	l.sugarLogger.Info(args...)
}

func (l *ApiLogger) Infof(template string, args ...interface{}) {
	l.sugarLogger.Infof(template, args...)
}

func (l *ApiLogger) Warn(args ...interface{}) {
	l.sugarLogger.Warn(args...)
}

func (l *ApiLogger) Warnf(template string, args ...interface{}) {
	l.sugarLogger.Warnf(template, args...)
}

func (l *ApiLogger) Error(args ...interface{}) {
	l.sugarLogger.Error(args...)
}

func (l *ApiLogger) Errorf(template string, args ...interface{}) {
	l.sugarLogger.Errorf(template, args...)
}

func (l *ApiLogger) DPanic(args ...interface{}) {
	l.sugarLogger.DPanic(args...)
}

func (l *ApiLogger) DPanicf(template string, args ...interface{}) {
	l.sugarLogger.DPanicf(template, args...)
}

func (l *ApiLogger) Panic(args ...interface{}) {
	l.sugarLogger.Panic(args...)
}

func (l *ApiLogger) Panicf(template string, args ...interface{}) {
	l.sugarLogger.Panicf(template, args...)
}

func (l *ApiLogger) Fatal(args ...interface{}) {
	l.sugarLogger.Fatal(args...)
}

func (l *ApiLogger) Fatalf(template string, args ...interface{}) {
	l.sugarLogger.Fatalf(template, args...)
}

/*
func (l *ApiLogger) Chronometer(mensagem string, inicio *time.Time) {
	fim := time.Now()
	tempo_execucao := fim.Sub(*inicio)
	if tempo_execucao == 0 {
		l.sugarLogger.DPanic("Chronometer -> ", mensagem, " : ", tempo_execucao.Milliseconds(), " millesegundos")
		return
	} else {
		l.sugarLogger.DPanic(mensagem, " : ", tempo_execucao.Seconds(), " segundos")
	}

}
*/

func (l *ApiLogger) Chronometer1(mensagem string, inicio *time.Time) {
	fim := time.Now()
	tempo_execucao := fim.Sub(*inicio)
	if tempo_execucao >= time.Duration(Hour) {
		l.sugarLogger.DPanic(mensagem, " : ", tempo_execucao.Hours(), " h")
		// fmt.Println(mensagem, " : ", tempo_execucao.Seconds(), " s")
	}
	if tempo_execucao < time.Duration(Hour) && tempo_execucao >= time.Duration(Minute) {
		l.sugarLogger.DPanic(mensagem, " : ", tempo_execucao.Seconds(), " min - minutos")
		// fmt.Println(mensagem, " : ", tempo_execucao.Seconds(), " s")
	}
	if tempo_execucao < time.Duration(Minute) && tempo_execucao >= time.Duration(Second) {
		l.sugarLogger.DPanic(mensagem, " : ", tempo_execucao.Seconds(), " s - segundos")
		// fmt.Println(mensagem, " : ", tempo_execucao.Seconds(), " s")
	}
	if tempo_execucao < time.Duration(Second) && tempo_execucao >= time.Duration(Millisecond) {
		l.sugarLogger.DPanic(mensagem, " : ", tempo_execucao.Milliseconds(), " ms -  Milessegundos")
		// fmt.Println(mensagem, " : ", tempo_execucao.Seconds(), " ms")
	}
	if tempo_execucao < time.Duration(Millisecond) && tempo_execucao >= time.Duration(Microsecond) {
		l.sugarLogger.DPanic(mensagem, " : ", tempo_execucao.Microseconds(), " us - Microssegundos")
		// fmt.Println(mensagem, " : ", tempo_execucao.Seconds(), " ms")
	}
	if tempo_execucao < time.Duration(Microsecond) && tempo_execucao >= time.Duration(0.000000000000000) {
		l.sugarLogger.DPanic(mensagem, " : ", tempo_execucao.Nanoseconds(), " ns - Nanossegundos")
		// fmt.Println(mensagem, " : ", tempo_execucao.Seconds(), " ns")
	}

}

func (l *ApiLogger) Chronometer(mensagem string, inicio *time.Time) {
	fim := time.Now()
	tempo_execucao := fim.Sub(*inicio)
	if tempo_execucao >= time.Duration(Hour) {
		l.sugarLogger.DPanic(mensagem, " : ", tempo_execucao.Seconds(), " segundos.")
	}
	if tempo_execucao < time.Duration(Hour) && tempo_execucao >= time.Duration(Minute) {
		l.sugarLogger.DPanic(mensagem, " : ", tempo_execucao.Seconds(), " segundos.")
	}
	if tempo_execucao < time.Duration(Minute) && tempo_execucao >= time.Duration(Second) {
		l.sugarLogger.DPanic(mensagem, " : ", tempo_execucao.Seconds(), " segundos.")
	}
	if tempo_execucao < time.Duration(Second) && tempo_execucao >= time.Duration(Millisecond) {
		l.sugarLogger.DPanic(mensagem, " : ", tempo_execucao.Seconds(), " segundos.")
	}
	if tempo_execucao < time.Duration(Millisecond) && tempo_execucao >= time.Duration(Microsecond) {
		l.sugarLogger.DPanic(mensagem, " : ", tempo_execucao.Seconds(), " segundos.")
	}
	if tempo_execucao < time.Duration(Microsecond) && tempo_execucao >= time.Duration(0.000000000000000) {
		l.sugarLogger.DPanic(mensagem, " : ", tempo_execucao.Seconds(), " segundos.")
	}

}
