package logger

import (
	"io"
	"log"
	"os"
)

const (
	debugLevel = iota
	infoLevel
	warningLevel
	errorLevel
	logLevelsCount // actually not a real log level, but simplifies some code
)

var loggers [logLevelsCount]*log.Logger
var controllerWriter io.Writer

func init() {
	// Start with only error lvel to stderr
	Setup(errorLevel, os.Stderr)
}

func logLevelString(level int) string {
	switch level {
	case debugLevel:
		return "DEBUG"
	case infoLevel:
		return "INFO"
	case warningLevel:
		return "WARNING"
	case errorLevel:
		return "ERROR"
	default:
		return "?????"
	}
}

func logLevelPrefix(level int) string {
	switch level {
	case debugLevel:
		return "[DBG] "
	case infoLevel:
		return "[INF] "
	case warningLevel:
		return "[WRN] "
	case errorLevel:
		return "[ERR] "
	default:
		return "[???] "
	}
}

func SetControllerWriter(w io.Writer) {
	controllerWriter = w
}

func Setup(level int, w ...io.Writer) {
	nullWriter := &nullWritter{}

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
				loggers[i] = log.New(makeWriters(append(w,
					&controllerLogger{wr: controllerWriter, level: logLevelString(i)})...),
					logLevelPrefix(i), log.Ldate|log.Ltime)
			} else {
				loggers[i] = log.New(makeWriters(w...), logLevelPrefix(i), log.Ldate|log.Ltime)
			}
		} else {
			loggers[i] = log.New(nullWriter, "", log.Ldate|log.Ltime)
		}
	}
}

func Debug() *log.Logger {
	return loggers[debugLevel]
}

func Info() *log.Logger {
	return loggers[infoLevel]
}

func Warning() *log.Logger {
	return loggers[warningLevel]
}

func Error() *log.Logger {
	return loggers[errorLevel]
}
