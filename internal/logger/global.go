package logger

import (
	"io"
	"log"
	"os"
)

var global *Logger

func init() {
	// Start with error+warning level to stderr
	SetupGlobalLoger(WarningLevel, os.Stderr)
}

func SetupGlobalLoger(level int, writers ...io.Writer) {
	global = New(level, writers...)
}

func Debug() *log.Logger {
	return global.loggers[DebugLevel]
}

func Info() *log.Logger {
	return global.loggers[InfoLevel]
}

func Warning() *log.Logger {
	return global.loggers[WarningLevel]
}

func Error() *log.Logger {
	return global.loggers[ErrorLevel]
}
