/**
 * Copyright 2021 Yuji Ito <llamerada.jp@gmail.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package logger

import (
	"fmt"
	"runtime"
)

type Log interface {
	Fatal(...interface{})
	Fatalf(string, ...interface{})
	Fatalln(...interface{})
	Error(...interface{})
	Errorf(string, ...interface{})
	Errorln(...interface{})
	Warn(...interface{})
	Warnf(string, ...interface{})
	Warnln(...interface{})
	Info(...interface{})
	Infof(string, ...interface{})
	Infoln(...interface{})
	Verbose(...interface{})
	Verbosef(string, ...interface{})
	Verboseln(...interface{})
}

type logImpl struct {
	logger Logger
}

func NewLog(logger Logger) Log {
	return &logImpl{
		logger: logger,
	}
}

func (l *logImpl) Fatal(a ...interface{}) {
	l.logger.Fatal(l.formatter(fmt.Sprint(a...)))
}

func (l *logImpl) Fatalf(format string, a ...interface{}) {
	l.logger.Fatal(l.formatter(fmt.Sprintf(format, a...)))
}

func (l *logImpl) Fatalln(a ...interface{}) {
	l.logger.Fatal(l.formatter(fmt.Sprintln(a...)))
}

func (l *logImpl) Error(a ...interface{}) {
	l.logger.Error(l.formatter(fmt.Sprint(a...)))
}

func (l *logImpl) Errorf(format string, a ...interface{}) {
	l.logger.Error(l.formatter(fmt.Sprintf(format, a...)))
}

func (l *logImpl) Errorln(a ...interface{}) {
	l.logger.Error(l.formatter(fmt.Sprintln(a...)))
}

func (l *logImpl) Warn(a ...interface{}) {
	l.logger.Warning(l.formatter(fmt.Sprint(a...)))
}

func (l *logImpl) Warnf(format string, a ...interface{}) {
	l.logger.Warning(l.formatter(fmt.Sprintf(format, a...)))
}

func (l *logImpl) Warnln(a ...interface{}) {
	l.logger.Warning(l.formatter(fmt.Sprintln(a...)))
}

func (l *logImpl) Info(a ...interface{}) {
	l.logger.Info(l.formatter(fmt.Sprint(a...)))
}

func (l *logImpl) Infof(format string, a ...interface{}) {
	l.logger.Info(l.formatter(fmt.Sprintf(format, a...)))
}

func (l *logImpl) Infoln(a ...interface{}) {
	l.logger.Info(l.formatter(fmt.Sprintln(a...)))
}

func (l *logImpl) Verbose(a ...interface{}) {
	l.logger.Verbose(l.formatter(fmt.Sprint(a...)))
}

func (l *logImpl) Verbosef(format string, a ...interface{}) {
	l.logger.Verbose(l.formatter(fmt.Sprintf(format, a...)))
}

func (l *logImpl) Verboseln(a ...interface{}) {
	l.logger.Verbose(l.formatter(fmt.Sprintln(a...)))
}

func (l *logImpl) formatter(message string) string {
	_, file, line, ok := runtime.Caller(2)
	if ok {
		return fmt.Sprintf("%s:%d %s", file, line, message)
	}
	return message
}
