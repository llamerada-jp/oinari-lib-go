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
package wasm

import (
	"syscall/js"

	"github.com/llamerada-jp/oinari-lib-go/pkg/logger"
)

type loggerImpl struct {
	logger js.Value
}

func NewLogger() logger.Logger {
	return &loggerImpl{
		logger: js.Global().Get("logger"),
	}
}

func (l *loggerImpl) Fatal(message string) {
	l.logger.Call("fatal", message)
}

func (l *loggerImpl) Error(message string) {
	l.logger.Call("error", message)
}

func (l *loggerImpl) Warning(message string) {
	l.logger.Call("warn", message)
}

func (l *loggerImpl) Info(message string) {
	l.logger.Call("info", message)
}

func (l *loggerImpl) Verbose(message string) {
	l.logger.Call("verbose", message)
}
