package util

import (
	"fmt"
	"log"
	"os"
)

const (
	DEBUG = 1 + iota
	INFO
	WARNING
	ERROR
	FATAL
	PANIC
)

var logger = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
var logLevel = INFO

func InitLogger(logPath string, logLevel int) {
	if logPath != "" {
		file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			LogError("failed open file: %s, %s", logPath, err)
		} else {
			logger = log.New(file, "", log.LstdFlags|log.Lshortfile)
		}
	}
	if logLevel >= DEBUG && logLevel <= PANIC {
		logLevel = logLevel
	}
}

func LogPanic(format string, v ...interface{}) {
	if logLevel <= PANIC {
		format = "[PANIC] " + format
		s := fmt.Sprintf(format, v...)
		logger.Output(2, s)
		panic(s)
	}
}

func LogFatal(format string, v ...interface{}) {
	if logLevel <= FATAL {
		format = "[FATAL] " + format
		logger.Output(2, fmt.Sprintf(format, v...))
		os.Exit(1)
	}
}

func LogError(format string, v ...interface{}) {
	if logLevel <= ERROR {
		format = "[ERROR] " + format
		logger.Output(2, fmt.Sprintf(format, v...))
	}
}

func LogWarn(format string, v ...interface{}) {
	if logLevel <= WARNING {
		format = "[WARNING] " + format
		logger.Output(2, fmt.Sprintf(format, v...))
	}
}

func LogInfof(format string, v ...interface{}) {
	if logLevel <= INFO {
		format = "[INFO] " + format
		logger.Output(2, fmt.Sprintf(format, v...))
	}
}

func LogInfo(v ...interface{}) {
	if logLevel <= INFO {
		s := "[INFO] " + fmt.Sprintln(v...)
		logger.Output(2, s)
	}
}

func LogDebugf(format string, v ...interface{}) {
	if logLevel <= DEBUG {
		format = "[DEBUG] " + format
		logger.Output(2, fmt.Sprintf(format, v...))
	}
}

func LogDebug(v ...interface{}) {
	if logLevel <= DEBUG {
		s := "[DEBUG] " + fmt.Sprintln(v...)
		logger.Output(2, s)
	}
}
