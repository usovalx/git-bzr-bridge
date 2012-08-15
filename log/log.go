//
// Simple wrapper around standard Go Logger
// adding Logger names and levels
//
package log

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
)

// Logging levels
type logLevel int

const (
	DEBUG logLevel = iota
	INFO
	ERROR
	PANIC
	NONE
)

var levels = [...]string{
	"DEBUG",
	"INFO",
	"ERROR",
	"PANIC",
	"NONE",
}

func (l logLevel) String() string { return levels[l] }

// Only log messages with level equal or higher to the specified one will be shown.
// PANIC messages are handled separately, and will be shown on the 
// stderr even if MinLogLevel is set to NONE
var MinLogLevel = INFO

// actual Loggers -- normal one and the one for PANIC messages
var std = log.New(os.Stdout, "", log.LstdFlags)
var err = log.New(os.Stderr, "", log.LstdFlags)

// A Logger represents active logging object with given name
type Logger struct {
	name string
}

// Create new Logger with the given name
func New(name string) *Logger {
	return &Logger{name}
}

// Print DEBUG level log line. Arguments are handled in the manner of fmt.Print
func (l *Logger) Debug(v ...interface{}) {
	l.log(DEBUG, 2, fmt.Sprint(v...))
}

// Print DEBUG level log line. Arguments are handled in the manner of fmt.Println
func (l *Logger) Debugln(v ...interface{}) {
	l.log(DEBUG, 2, fmt.Sprintln(v...))
}

// Print DEBUG level log line. Arguments are handled in the manner of fmt.Printf
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.log(DEBUG, 2, fmt.Sprintf(format, v...))
}

// Print INFO level log line. Arguments are handled in the manner of fmt.Print
func (l *Logger) Info(v ...interface{}) {
	l.log(INFO, 2, fmt.Sprint(v...))
}

// Print INFO level log line. Arguments are handled in the manner of fmt.Println
func (l *Logger) Infoln(v ...interface{}) {
	l.log(INFO, 2, fmt.Sprintln(v...))
}

// Print INFO level log line. Arguments are handled in the manner of fmt.Printf
func (l *Logger) Infof(format string, v ...interface{}) {
	l.log(INFO, 2, fmt.Sprintf(format, v...))
}

// Print ERROR level log line. Arguments are handled in the manner of fmt.Print
func (l *Logger) Error(v ...interface{}) {
	l.log(ERROR, 2, fmt.Sprint(v...))
}

// Print ERROR level log line. Arguments are handled in the manner of fmt.Println
func (l *Logger) Errorln(v ...interface{}) {
	l.log(ERROR, 2, fmt.Sprintln(v...))
}

// Print ERROR level log line. Arguments are handled in the manner of fmt.Printf
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.log(ERROR, 2, fmt.Sprintf(format, v...))
}

// Print PANIC level log line. Arguments are handled in the manner of fmt.Print
// After log message is written it calls panic()
func (l *Logger) Panic(v ...interface{}) {
	msg := fmt.Sprint(v...)
	l.log(PANIC, 2, msg)
	panic(msg)
}

// Print PANIC level log line. Arguments are handled in the manner of fmt.Println
// After log message is written it calls panic()
func (l *Logger) Panicln(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	l.log(PANIC, 2, msg)
	panic(msg)
}

// Print PANIC level log line. Arguments are handled in the manner of fmt.Printf
// After log message is written it calls panic()
func (l *Logger) Panicf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.log(PANIC, 2, msg)
	panic(msg)
}

// Write log message
func (l *Logger) log(level logLevel, calldepth int, msg string) {
	if level >= MinLogLevel && level < PANIC {
		std.Printf("- %-5s - %s: %s", level, l.name, msg)
	} else if level >= PANIC {
		// get file name & line number of where log message was produced
		_, file, line, ok := runtime.Caller(calldepth)
		if ok {
			file = path.Base(file)
		} else {
			file = "???"
			line = 0
		}
		err.Printf("- %-5s - %s(%s:%d): %s", level, l.name, file, line, msg)
	}
}
