// Package log implements a structured JSON log which can't be
// easily dependency injected into your microservice (on purpose).
//
// To use, override the package-scoped variables at runtime.
//
// This code may be copied and pasted into your microservice
// and modified to your liking. Put it in a package called
// log. A little copying is better than a little dependency.
//
package log

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

var (
	// Service name
	Service = ""

	// Time is your time function. Default is a millisecond timestamp.
	Time = func() interface{} {
		return time.Now().UnixNano() / int64(time.Millisecond)
	}

	// Tags are global static fields to publish for this process on
	// all log levels and callers
	Tags = fields{}

	// Default is the level used when calling Printf and Fatalf
	Default = Info
)

var (
	// Info, Warn, and so forth are commonly encountered log "levels".
	Info  = line{Level: "info"}
	Warn  = line{Level: "warn"}
	Error = line{Level: "error"}
	Fatal = line{Level: "fatal"}

	// Debug is a special level, it is only printed if DebugOn is true
	Debug   = line{Level: "debug"}
	DebugOn = false
)

var stderr = io.Writer(os.Stderr)

// Printf and Fatalf exist to make this package somewhat compatible with
// the go standard log.
func Printf(f string, v ...interface{}) { Default.F(f, v...) }
func Fatalf(f string, v ...interface{}) { Fatal.F(f, v...) }

// SetOutput sets the log output to w. It returns the previous writer used.
func SetOutput(w io.Writer) (old io.Writer) {
	old = stderr
	stderr = w
	return old
}

type line struct {
	fields
	Level string
	msg   string
}

// Printf attaches the formatted message to line and outputs
// the result to Stderr. Callers should call F() when not adding
// extra fields explicitly.
//
// The following fields are pre-declared, and emitted in order:
// (1) svc: value of Service
// (2) time: result of calling Time()
// (3) level: the log level
// (4) msg: the formatted string provided to Printf
//
// Prefer log.Error.F() to log.Error.Printf() unless using Add
func (l line) Printf(f string, v ...interface{}) {
	if l.Level == Debug.Level && !DebugOn {
		return
	}
	fmt.Fprintln(stderr, l.Msg(f, v...).String())
	if l.Level == "fatal" {
		panic("fatal log level")
	}
}

// F is equivalent to Printf
func (l line) F(f string, v ...interface{}) {
	l.Printf(f, v...)
}

// Msg returns a copy of l with the msg field set
// to the formatted string argument provided. Most
// callers should use the l.Printf or l.F
func (l line) Msg(f string, v ...interface{}) line {
	l.msg = fmt.Sprintf(f, v...)
	return l
}

// String returns the line as a string
func (l line) String() string {
	hdr := append(fields{
		"svc", Service,
		"time", Time(),
		"level", l.Level,
	}, Tags...)
	hdr = append(hdr, l.fields...)
	return append(hdr, "msg", l.msg).String()
}

// Add returns a copy of the line with the custom fields provided
// fields should be provided in pairs, otherwise they are ignored:
//
// Info.Add("railway", "east", "stop", 5).Printf("train stopped")
//
// Add always makes a deep copy.
func (l line) Add(field ...interface{}) line {
	l.fields = l.fields.Add(field...)
	return l
}

// New returns a log line with an extra field list
func New(fields ...interface{}) line {
	return Default.Add(fields...)
}

// Info and the rest of these convert l into another log level
func (l line) Info() line  { l.Level = Info.Level; return l }
func (l line) Error() line { l.Level = Error.Level; return l }
func (l line) Warn() line  { l.Level = Warn.Level; return l }
func (l line) Fatal() line { l.Level = Fatal.Level; return l }

type fields []interface{}

func (f fields) String() (s string) {
	sep := ""
	for i := 0; i+1 < len(f); i += 2 {
		key, val := f[i], f[i+1]
		if val == "" || val == nil {
			continue
		}
		s += fmt.Sprintf(`%s%q:%s`, sep, key, quote(val))
		sep = ", "
	}
	return "{" + s + "}"
}

func (l fields) Add(f ...interface{}) fields {
	return append(append(fields{}, l...), f...)
}

func quote(v interface{}) string {
	if v == nil {
		v = ""
	}
	switch v.(type) {
	case fmt.Stringer, error:
		v = fmt.Sprint(v)
	}
	data, _ := json.Marshal(v)
	return string(data)
}
