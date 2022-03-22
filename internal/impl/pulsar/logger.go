package pulsar

import (
	plog "github.com/apache/pulsar-client-go/pulsar/log"

	"github.com/benthosdev/benthos/v4/public/service"
)

// DefaultLogger returns a logger that wraps Benthos Modular logger.
func createDefaultLogger(l *service.Logger) plog.Logger {
	return defaultLogger{
		backend: l,
	}
}

type defaultLogger struct {
	backend *service.Logger
}

func (l defaultLogger) SubLogger(fields plog.Fields) plog.Logger {
	return l
}

func (l defaultLogger) WithFields(fields plog.Fields) plog.Entry {
	return l
}

func (l defaultLogger) WithField(name string, value interface{}) plog.Entry {
	return l
}

func (l defaultLogger) WithError(err error) plog.Entry {
	return l
}

func (l defaultLogger) Debug(args ...interface{}) {
	l.backend.Debugf("%v", args)
}

func (l defaultLogger) Info(args ...interface{}) {
	l.backend.Infof("%v", args)
}

func (l defaultLogger) Warn(args ...interface{}) {
	l.backend.Warnf("%v", args)
}

func (l defaultLogger) Error(args ...interface{}) {
	l.backend.Errorf("%v", args)
}

func (l defaultLogger) Debugf(format string, args ...interface{}) {
	l.backend.Debugf(format, args)
}

func (l defaultLogger) Infof(format string, args ...interface{}) {
	l.backend.Infof(format, args)
}

func (l defaultLogger) Warnf(format string, args ...interface{}) {
	l.backend.Warnf(format, args)
}

func (l defaultLogger) Errorf(format string, args ...interface{}) {
	l.backend.Errorf(format, args)
}

// NoopLogger returns a logger that does nothing.
func NoopLogger() plog.Logger {
	return noopLogger{}
}

type noopLogger struct{}

func (n noopLogger) SubLogger(fields plog.Fields) plog.Logger {
	return n
}

func (n noopLogger) WithFields(fields plog.Fields) plog.Entry {
	return n
}
func (n noopLogger) WithField(name string, value interface{}) plog.Entry {
	return n
}
func (n noopLogger) WithError(err error) plog.Entry {
	return n
}

func (n noopLogger) Debug(args ...interface{}) {}
func (n noopLogger) Info(args ...interface{})  {}
func (n noopLogger) Warn(args ...interface{})  {}
func (n noopLogger) Error(args ...interface{}) {}

func (n noopLogger) Debugf(format string, args ...interface{}) {}
func (n noopLogger) Infof(format string, args ...interface{})  {}
func (n noopLogger) Warnf(format string, args ...interface{})  {}
func (n noopLogger) Errorf(format string, args ...interface{}) {}
