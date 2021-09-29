// logger - log level and controller logging package.

// This part is a bit tricky
// Actually in application there will be 2 loggers:
//  * a Full-featured global loger
//  * a controller logger.
// Global logger logs to stdout (can be configured log to file easily)
// Global logger sends log to remote controller as well.
// And here is why I need a separate controller logger:
// If I use global logger from controller package -
// it may result in recursive logging - a logging from controller package would also
// be sent to remote controller logging.
// And we get a recursion here, which  will deadlock on Write function
// Thus I will create a separate local logger instance, without remote controller Writer.
// And I can log controller package errors locally
// (if controller package errors happen - then most probably I may not log them back to
// remote controller, so at least have some errors logged locally).

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

// Yes this API is not perfect.
// But I made controller Writer the first parameter
// So this way compiler will catch parameter errors at compile time.
// In a perfect world it would be best to detect controllerWriter inside
// But this leads to circular dependencies.
// Hope to find a more pretty solution one day.
func New(controller io.Writer, level int, w ...io.Writer) *Logger {
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
			if controller != nil {
				lgr.loggers[i] = log.New(makeWriters(append(w,
					&controllerLogger{wr: controller, level: logLevelString(i)})...),
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
