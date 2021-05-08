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

import "log"

type stdlog struct {
}

func LoggerWithStdLog() Logger {
	return &stdlog{}
}

func (s *stdlog) Fatal(message string) {
	log.Fatalf("[FATAL] %s", message)
}

func (s *stdlog) Error(message string) {
	log.Printf("[ERROR] %s", message)
}

func (s *stdlog) Warning(message string) {
	log.Printf("[WARNING] %s", message)
}

func (s *stdlog) Info(message string) {
	log.Printf("[INFO] %s", message)
}

func (s *stdlog) Verbose(message string) {
	log.Printf("[VERBOSE] %s", message)
}
