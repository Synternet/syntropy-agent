package logger

import (
	"io"
	"log"
)

const (
	DebugLevel = iota
	InfoLevel
	WarningLevel
	ErrorLevel
	logLevelsCount // actually not a real log level, but simplifies some code
)

type Logger struct {
	loggers [logLevelsCount]*log.Logger
}

func logLevelString(level int) string {
	switch level {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarningLevel:
		return "WARNING"
	case ErrorLevel:
		return "ERROR"
	default:
		return "?????"
	}
}

func logLevelPrefix(level int) string {
	switch level {
	case DebugLevel:
		return "[DBG] "
	case InfoLevel:
		return "[INF] "
	case WarningLevel:
		return "[WRN] "
	case ErrorLevel:
		return "[ERR] "
	default:
		return "[???] "
	}
}

func New(level int, writers ...io.Writer) *Logger {
	var controllerWriter io.Writer
	w := []io.Writer{}
	for _, onewriter := range writers {
		// Controller logger is a special case, because the logger itself must know its log level
		// So here I am doing a smart logger type setection and sorting them
		// to generic loggers slice and a controller logger (should be only one instance in variadic parameters)
		switch typewr := onewriter.(type) {
		case *controllerLogger:
			controllerWriter = typewr
		default:
			w = append(w, typewr)
		}
	}

	nullWriter := &nullWritter{}
	lgr := Logger{}

	makeWriters := func(wrs ...io.Writer) io.Writer {

		var writers io.Writer

		switch {
		case wrs == nil:
			writers = nullWriter
		case len(wrs) == 0:
			writers = nullWriter
		case len(wrs) == 1:
			writers = wrs[0]
		default:
			writers = io.MultiWriter(wrs...)
		}
		return writers
	}

	for i := 0; i < logLevelsCount; i++ {
		if i >= level {
			if controllerWriter != nil {
				lgr.loggers[i] = log.New(makeWriters(append(w,
					&controllerLogger{wr: controllerWriter, level: logLevelString(i)})...),
					logLevelPrefix(i), log.Ldate|log.Ltime)
			} else {
				lgr.loggers[i] = log.New(makeWriters(w...), logLevelPrefix(i), log.Ldate|log.Ltime)
			}
		} else {
			lgr.loggers[i] = log.New(nullWriter, "", log.Ldate|log.Ltime)
		}
	}
	return &lgr
}

func (lgr *Logger) Debug() *log.Logger {
	return lgr.loggers[DebugLevel]
}

func (lgr *Logger) Info() *log.Logger {
	return lgr.loggers[InfoLevel]
}

func (lgr *Logger) Warning() *log.Logger {
	return lgr.loggers[WarningLevel]
}

func (lgr *Logger) Error() *log.Logger {
	return lgr.loggers[ErrorLevel]
}
