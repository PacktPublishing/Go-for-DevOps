/*
Package log is a replacement for the standard library log package that
logs to OTEL spans contained in Context objects. These are seen as
events with the attribute "log" set to true.

The preferred way to log is to use an event:
	func someFunc(ctx context.Context) {
		e := NewEvent("someFunc()")
		defer e.Done(ctx)
		start := time.Now()
		defer func() {
			e.Add("latency.ns", int(time.Since(start)))
		}()
	}

This records an event in the current span that has a key of "latency.ns" with the value in nano-seconds the operation took.

You can use this to log in a similar manner to the logging package with Println and Printf.  This is generally only useful for some generic debugging where you want to log something and filter the trace by messages with key "log". Generally these are messages you don't want to keep.
	func main() {
		ctx := context.Background()

		log.SetFlags(log.LstdFlags | log.Lshortfile)

		log.Println(ctx, "Starting main")

		log.Printf(ctx, "Env variables: %v", os.Environ())
	}

The above won't log anything, as there is no Span on the Context. If there
was it would get output to the Open Telementry provider.

If you need to use the standard library log, you can use Logger:
	log.Logger.Println("hello world")

This would print whever the stanard logger prints to. This defaults
to the standard logger, but you can replace with another Logger if you wish.

You should only log messages with a standard logger when it can't be output to a trace. These are critical messages that indicate a definite bug. This keeps logging to only critical events and de-clutters what you need to look at to when doing a debug.
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

// setup the standard logger with flags.
var std = &logger{flag: LstdFlags}

// pool provides a pool of Event objects to keep our allocations to a minimum.
var pool = &eventPool{
	buf: make(chan *Event, 100),
	pool: sync.Pool{
		New: func() interface{} {
			return &Event{}
		},
	},
}

// eventPool uses a set amount of Event objects and a sync.Pool for overflow.
// Note: this would actually be a great place for metrics to key in on what would be
// an optimal size for buf to prevent pool use.
type eventPool struct {
	buf  chan *Event
	pool sync.Pool
}

func (e *eventPool) get() *Event {
	select {
	case ev := <-e.buf:
		return ev
	default:
	}
	return e.pool.Get().(*Event)
}

func (e *eventPool) put(ev *Event) {
	ev.reset()
	select {
	case e.buf <- ev:
	default:
	}
	e.pool.Put(ev)
}

// Event represents a named event that occurs. This is the prefered way to log data.
// Events have attributes and those attributes are key/value pairs. You create
// an event and stuff attributes using Add() until the event is over and call Done().
// This will render the event to the current span. if no attrs exist, the event is ignored.
// To avoid extra allocations
type Event struct {
	name  string
	attrs []attribute.KeyValue
}

// NewEvent returns a new Event.
func NewEvent(name string) *Event {
	ev := pool.get()
	ev.name = name
	return ev
}

func (e *Event) reset() {
	e.name = ""
	e.attrs = e.attrs[0:0]
}

// Add adds an attribute named k with value i. i can be: bool, []bool, float64, []float64, int, []int, int64, []int64, string and []string.
// If the value isn't one of those values, a standard log message is printed indicating a bug.
func (e *Event) Add(k string, i interface{}) {
	if e.name == "" {
		return
	}
	switch v := i.(type) {
	case bool:
		e.attrs = append(e.attrs, attribute.Bool(k, v))
	case []bool:
		e.attrs = append(e.attrs, attribute.BoolSlice(k, v))
	case float64:
		e.attrs = append(e.attrs, attribute.Float64(k, v))
	case []float64:
		e.attrs = append(e.attrs, attribute.Float64Slice(k, v))
	case int:
		e.attrs = append(e.attrs, attribute.Int(k, v))
	case []int:
		e.attrs = append(e.attrs, attribute.IntSlice(k, v))
	case int64:
		e.attrs = append(e.attrs, attribute.Int64(k, v))
	case []int64:
		e.attrs = append(e.attrs, attribute.Int64Slice(k, v))
	case string:
		e.attrs = append(e.attrs, attribute.String(k, v))
	case []string:
		e.attrs = append(e.attrs, attribute.StringSlice(k, v))
	case time.Duration:
		e.attrs = append(e.attrs, attribute.String(k, v.String()))
	default:
		log.Printf("bug: event.Add(): receiveing %T which is not supported", v)
	}
}

// Done renders the Event to the span in the Context. If there are no attributes on the Event, this is a no-oop.
// Once Done is called, the Event object MUST not be used again.
func (e *Event) Done(ctx context.Context) {
	defer pool.put(e)

	if e.name == "" {
		return
	}
	span := trace.SpanFromContext(ctx)
	if e.attrs == nil {
		return
	}
	span.AddEvent(e.name, trace.WithAttributes(e.attrs...))
}

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
