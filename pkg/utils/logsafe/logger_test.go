package logsafe

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

type capture struct{ level, text string }

func (c *capture) Trace(v ...interface{}) { c.level = "Trace"; c.text = fmt.Sprint(v...) }
func (c *capture) Tracef(f string, v ...interface{}) {
	c.level = "Trace"
	c.text = fmt.Sprintf(f, v...)
}
func (c *capture) Debug(v ...interface{}) { c.level = "Debug"; c.text = fmt.Sprint(v...) }
func (c *capture) Debugf(f string, v ...interface{}) {
	c.level = "Debug"
	c.text = fmt.Sprintf(f, v...)
}
func (c *capture) Info(v ...interface{})            { c.level = "Info"; c.text = fmt.Sprint(v...) }
func (c *capture) Infof(f string, v ...interface{}) { c.level = "Info"; c.text = fmt.Sprintf(f, v...) }
func (c *capture) Warn(v ...interface{})            { c.level = "Warn"; c.text = fmt.Sprint(v...) }
func (c *capture) Warnf(f string, v ...interface{}) { c.level = "Warn"; c.text = fmt.Sprintf(f, v...) }
func (c *capture) Error(v ...interface{})           { c.level = "Error"; c.text = fmt.Sprint(v...) }
func (c *capture) Errorf(f string, v ...interface{}) {
	c.level = "Error"
	c.text = fmt.Sprintf(f, v...)
}

type forbiddenFormatter struct{}

func (forbiddenFormatter) String() string { panic("untrusted formatter invoked") }

func TestLoggerAllLevels(t *testing.T) {
	sink := &capture{}
	l := New(sink)
	formatted := []struct {
		name string
		call func(string, ...interface{})
	}{
		{"Trace", l.Tracef}, {"Debug", l.Debugf}, {"Info", l.Infof}, {"Warn", l.Warnf}, {"Error", l.Errorf},
	}
	for _, tt := range formatted {
		tt.call("close failed, clientID: %d, meta: %s, error: %v, object: %v", uint64(123), []byte(`{"token":"SECRET"}`), errors.New("SECRET"), forbiddenFormatter{})
		if sink.level != tt.name || strings.Contains(sink.text, "SECRET") || !strings.Contains(sink.text, "clientID: 123") {
			t.Fatal(sink)
		}
	}
	plain := []func(...interface{}){l.Trace, l.Debug, l.Info, l.Warn, l.Error}
	for _, call := range plain {
		call("SECRET", []byte("SECRET"), errors.New("SECRET"), forbiddenFormatter{})
		if strings.Contains(sink.text, "SECRET") {
			t.Fatal(sink.text)
		}
	}
}
func TestSafeArgsDoesNotMutateCaller(t *testing.T) {
	args := []interface{}{"SECRET", uint64(42), true, nil}
	got := safeArgs(args)
	if args[0] != "SECRET" || got[0] != omitted || got[1] != uint64(42) || got[2] != true || got[3] != nil {
		t.Fatal("argument isolation failed")
	}
}
