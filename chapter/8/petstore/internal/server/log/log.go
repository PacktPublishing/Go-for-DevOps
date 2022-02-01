/*
Package log is a replacement for the standard library log package that
logs to OTEL spans contained in Context objects. These are seen as
events with the attribute "log" set to true.

Usage is simlar to the standard log package:
	func main() {
		ctx := context.Background()

		log.SetFlags(log.LstdFlags | log.Lshortfile)

		log.Println(ctx, "Starting main")

		log.Printf(ctx, "Env variables: %v", os.Environ())
	}

The above won't log anything, as there is no Span on the Context. If there
was it would get output to the Open Telementry tracer.

If you need to use the standard library log, you can use Logger:
	log.Logger.Println("hello world")

This would print whever the stanard logger prints to. This defaults
to the standard logger, but you can replace with another Logger if you wish.
*/
package log

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	Ldate         = 1 << iota     // the date in the local time zone: 2009/01/23
	Ltime                         // the time in the local time zone: 01:23:23
	Lmicroseconds                 // microsecond resolution: 01:23:23.123123.  assumes Ltime.
	Llongfile                     // full file name and line number: /a/b/c/d.go:23
	Lshortfile                    // final file name element and line number: d.go:23. overrides Llongfile
	LUTC                          // if Ldate or Ltime is set, use UTC rather than the local time zone
	LstdFlags     = Ldate | Ltime // initial values for the standard logger
)

// Logger provides access to the standard library's default logger.
// This can be replaced in main with a logger of your choice.
var Logger *log.Logger = log.Default()

var std = &logger{flag: LstdFlags}

// Println acts like log.Println() except we log to the OTEL span in the Context.
func Println(ctx context.Context, v ...interface{}) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	std.output(span, 2, fmt.Sprintln(v...))
}

// Printf acts like log.Printf() except we log to the OTEL span in the Context.
func Printf(ctx context.Context, format string, v ...interface{}) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	std.output(span, 2, fmt.Sprintf(format, v...))
}

// SetFlags sets the output flags for the standard logger.
func SetFlags(flag int) {
	std.flag = flag
}

// logger is an implementation of log.Logger that writes to a Span.
type logger struct {
	mu   sync.Mutex
	flag int    // properties
	buf  []byte // for accumulating text to write
}

// Cheap integer to fixed-width decimal ASCII. Give a negative width to avoid zero-padding.
func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

func (l *logger) output(span trace.Span, calldepth int, s string) error {
	now := time.Now() // get this early
	var file string
	var line int
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.flag&(Lshortfile|Llongfile) != 0 {
		// Release lock while getting caller info - it's expensive.
		l.mu.Unlock()
		var ok bool
		_, file, line, ok = runtime.Caller(calldepth)
		if !ok {
			file = "???"
			line = 0
		}
		l.mu.Lock()
	}
	l.buf = l.buf[:0]
	l.formatHeader(&l.buf, now, file, line)
	l.buf = append(l.buf, s...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		l.buf = append(l.buf, '\n')
	}
	span.AddEvent(string(l.buf), trace.WithAttributes(attribute.Bool("log", true)))
	return nil
}

// formatHeader writes log header to buf in following order:
//   * date and/or time (if corresponding flags are provided),
//   * file and line number (if corresponding flags are provided),
func (l *logger) formatHeader(buf *[]byte, t time.Time, file string, line int) {
	if l.flag&(Ldate|Ltime|Lmicroseconds) != 0 {
		if l.flag&LUTC != 0 {
			t = t.UTC()
		}
		if l.flag&Ldate != 0 {
			year, month, day := t.Date()
			itoa(buf, year, 4)
			*buf = append(*buf, '/')
			itoa(buf, int(month), 2)
			*buf = append(*buf, '/')
			itoa(buf, day, 2)
			*buf = append(*buf, ' ')
		}
		if l.flag&(Ltime|Lmicroseconds) != 0 {
			hour, min, sec := t.Clock()
			itoa(buf, hour, 2)
			*buf = append(*buf, ':')
			itoa(buf, min, 2)
			*buf = append(*buf, ':')
			itoa(buf, sec, 2)
			if l.flag&Lmicroseconds != 0 {
				*buf = append(*buf, '.')
				itoa(buf, t.Nanosecond()/1e3, 6)
			}
			*buf = append(*buf, ' ')
		}
	}
	if l.flag&(Lshortfile|Llongfile) != 0 {
		if l.flag&Lshortfile != 0 {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
		}
		*buf = append(*buf, file...)
		*buf = append(*buf, ':')
		itoa(buf, line, -1)
		*buf = append(*buf, ": "...)
	}
}
