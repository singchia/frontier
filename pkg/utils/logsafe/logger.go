// Package logsafe protects server-side SDK diagnostics from opaque peer metadata.
package logsafe

import "github.com/jumboframes/armorigo/log"

const omitted = "<redacted>"

// Logger retains trusted SDK format templates and numeric diagnostic identifiers.
// Arbitrary strings, errors and objects are opaque: their formatters must not run.
// Format strings must be static application code, never peer-provided text.
type Logger struct{ sink log.Logger }

func New(sink log.Logger) *Logger { return &Logger{sink: sink} }

var _ log.Logger = (*Logger)(nil)

func safeArgs(args []interface{}) []interface{} {
	out := make([]interface{}, len(args))
	for i, arg := range args {
		switch arg.(type) {
		case nil, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			out[i] = arg
		default:
			out[i] = omitted
		}
	}
	return out
}
func (l *Logger) Trace(args ...interface{})                 { l.sink.Trace(safeArgs(args)...) }
func (l *Logger) Tracef(format string, args ...interface{}) { l.sink.Tracef(format, safeArgs(args)...) }
func (l *Logger) Debug(args ...interface{})                 { l.sink.Debug(safeArgs(args)...) }
func (l *Logger) Debugf(format string, args ...interface{}) { l.sink.Debugf(format, safeArgs(args)...) }
func (l *Logger) Info(args ...interface{})                  { l.sink.Info(safeArgs(args)...) }
func (l *Logger) Infof(format string, args ...interface{})  { l.sink.Infof(format, safeArgs(args)...) }
func (l *Logger) Warn(args ...interface{})                  { l.sink.Warn(safeArgs(args)...) }
func (l *Logger) Warnf(format string, args ...interface{})  { l.sink.Warnf(format, safeArgs(args)...) }
func (l *Logger) Error(args ...interface{})                 { l.sink.Error(safeArgs(args)...) }
func (l *Logger) Errorf(format string, args ...interface{}) { l.sink.Errorf(format, safeArgs(args)...) }
