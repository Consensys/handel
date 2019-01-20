package handel

import (
	"os"

	"github.com/go-kit/kit/log"
	// conflicts with handel.level type
	lvl "github.com/go-kit/kit/log/level"
)

// Logger is a interface that can log to different levels. Handel calls these
// methods with key-value pairs as in structured logging framework do.
type Logger interface {
	Info(keyvals ...interface{})
	Debug(keyvals ...interface{})
	Warn(keyvals ...interface{})
	Error(keyvals ...interface{})
	// With returns a new Logger that inserts the given key value pairs for each
	// statements at each levels
	With(keyvals ...interface{}) Logger
}

// DefaultLevel is the default level where statements are logged. One can change
// this variable inside init() to change the default level, or construct
// explicitly a Logger.
var DefaultLevel = lvl.AllowInfo()

// DefaultLogger is the default logger that only outputs statemetns at the
// default level. One can change the DefaultLevel variable inside init() to
// change the default level output.
var DefaultLogger = NewKitLogger(DefaultLevel)

type kitLogger struct {
	log.Logger
}

// NewKitLoggerFrom returns a Logger out of a go-kit/kit/log logger interface. The
// caller can set the options that it needs to the logger first. By default, it
// wraps the logger with a SyncLogger since Handel is highly concurrent.
func NewKitLoggerFrom(l log.Logger) Logger {
	return &kitLogger{log.NewSyncLogger(l)}
}

// NewKitLogger returns a Logger based on go-kit/kit/log default logger
// structure that outputs to stdout. You can pass in options to only allow
// certain levels. By default, it also includes the caller stack.
func NewKitLogger(opts ...lvl.Option) Logger {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	for _, opt := range opts {
		logger = lvl.NewFilter(logger, opt)
	}
	logger = log.With(logger, "call", log.Caller(8))
	return NewKitLoggerFrom(logger)
}

func (k *kitLogger) Info(kv ...interface{}) {
	lvl.Info(k.Logger).Log(kv...)
}

func (k *kitLogger) Debug(kv ...interface{}) {
	lvl.Debug(k.Logger).Log(kv...)
}

func (k *kitLogger) Warn(kv ...interface{}) {
	lvl.Warn(k.Logger).Log(kv...)
}

func (k *kitLogger) Error(kv ...interface{}) {
	lvl.Error(k.Logger).Log(kv...)
}

func (k *kitLogger) With(kv ...interface{}) Logger {
	newLogger := log.With(k.Logger, kv...)
	return NewKitLoggerFrom(newLogger)
}
