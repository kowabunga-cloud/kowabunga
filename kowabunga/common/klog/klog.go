/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package klog

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/op/go-logging"
)

// LoggerConfiguration defines custom configuration of a logging engine
type LoggerConfiguration struct {
	Type    string
	Enabled bool
	Level   string
	File    string
}

var logger = logging.MustGetLogger("klog")

// Init initializes the custom logger sub-system
func Init(name string, loggers []LoggerConfiguration) {

	var logLevels = map[string]logging.Level{
		"CRITICAL": logging.CRITICAL,
		"ERROR":    logging.ERROR,
		"WARNING":  logging.WARNING,
		"NOTICE":   logging.NOTICE,
		"INFO":     logging.INFO,
		"DEBUG":    logging.DEBUG,
	}

	syslogLevels := map[string]syslog.Priority{
		"CRITICAL": syslog.LOG_CRIT,
		"ERROR":    syslog.LOG_ERR,
		"WARNING":  syslog.LOG_WARNING,
		"NOTICE":   syslog.LOG_NOTICE,
		"INFO":     syslog.LOG_INFO,
		"DEBUG":    syslog.LOG_DEBUG,
	}

	logger = logging.MustGetLogger(name)

	backends := make([]logging.Backend, 0, len(loggers))
	for _, l := range loggers {
		if !l.Enabled {
			continue
		}
		switch l.Type {
		case "console":
			// console logging
			_, present := logLevels[l.Level]
			if !present {
				log.Fatalln("Unsupported log-level value for logger: ", l.Type, l.Level)
			}
			consoleFmt := logging.MustStringFormatter(
				`%{color} ▶ [%{level:.4s} %{id:05x}%{color:reset}] %{message}`,
			)
			console := logging.NewLogBackend(os.Stdout, "", 0)
			consoleFormat := logging.NewBackendFormatter(console, consoleFmt)
			consoleLevel := logging.AddModuleLevel(consoleFormat)
			consoleLevel.SetLevel(logLevels[l.Level], "")
			backends = append(backends, consoleLevel)
		case "file":
			// file logging
			_, present := logLevels[l.Level]
			if !present {
				log.Fatalln("Unsupported log-level value for logger: ", l.Type, l.Level)
			}
			logfile, err := os.OpenFile(l.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if err != nil {
				log.Fatalln("Failed to open log file:", err)
			}
			fileFmt := logging.MustStringFormatter(
				`[%{time:2006-01-02 15:04:05.000}] [%{level:.4s}] [%{id:05x}] %{message}`,
			)
			file := logging.NewLogBackend(logfile, "", 0)
			fileFormat := logging.NewBackendFormatter(file, fileFmt)
			fileLevel := logging.AddModuleLevel(fileFormat)
			fileLevel.SetLevel(logLevels[l.Level], "")
			backends = append(backends, fileLevel)
		case "syslog":
			// syslog logging
			_, present := syslogLevels[l.Level]
			if !present {
				log.Fatalln("Unsupported log-level value for logger: ", l.Type, l.Level)
			}
			syslogLevel, _ := logging.NewSyslogBackendPriority(logger.Module, syslogLevels[l.Level])
			backends = append(backends, syslogLevel)
		}
	}

	// Set the backends to be used.
	logging.SetBackend(backends...)
}

func runtimePC(msg string) string {
	pc, file, line, ok := runtime.Caller(3)
	if !ok {
		return ""
	}

	filename := file[strings.LastIndex(file, "/")+1:] + ":" + strconv.Itoa(line)
	funcname := runtime.FuncForPC(pc).Name()
	fn := funcname[strings.LastIndex(funcname, ".")+1:]
	return fmt.Sprintf("[%s][%s()] %s", filename, fn, msg)
}

func prependPC(args ...interface{}) string {
	msg := fmt.Sprint(args...)
	return runtimePC(msg)
}

func prependPCf(format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)
	return runtimePC(msg)
}

// Critical logs a simple message when severity is set to CRITICAL or above
func Critical(args ...interface{}) {
	logger.Critical(prependPC(args...))
}

// Criticalf logs a formatted message when severity is set to CRITICAL or above
func Criticalf(format string, args ...interface{}) {
	logger.Criticalf(prependPCf(format, args...))
}

// Error logs a simple message when severity is set to ERROR or above
func Error(args ...interface{}) {
	logger.Error(prependPC(args...))
}

// Errorf logs a formatted message when severity is set to ERROR or above
func Errorf(format string, args ...interface{}) {
	logger.Errorf(prependPCf(format, args...))
}

// Warning logs a simple message when severity is set to WARNING or above
func Warning(args ...interface{}) {
	logger.Warning(args...)
}

// Warningf logs a formatted message when severity is set to WARNING or above
func Warningf(format string, args ...interface{}) {
	logger.Warningf(format, args...)
}

// Notice logs a simple message when severity is set to NOTICE or above
func Notice(args ...interface{}) {
	logger.Notice(args...)
}

// Noticef logs a formatted message when severity is set to NOTICE or above
func Noticef(format string, args ...interface{}) {
	logger.Noticef(format, args...)
}

// Info logs a simple message when severity is set to INFO or above
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Infof logs a formatted message when severity is set to INFO or above
func Infof(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

// Debug logs a simple message when severity is set to DEBUG or above
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

// Debugf logs a formatted message when severity is set to DEBUG or above
func Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}

// Fatal logs a simple message which is fatal
func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

// Fatalf logs a formatted message which is fatal
func Fatalf(format string, args ...interface{}) {
	logger.Fatalf(format, args...)
}

// Panic logs a simple message which leads to panic
func Panic(args ...interface{}) {
	logger.Panic(args...)
}

// Panicf logs a formatted message which leads to panic
func Panicf(format string, args ...interface{}) {
	logger.Panicf(format, args...)
}
