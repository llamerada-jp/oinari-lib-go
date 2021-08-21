// +build js

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
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/llamerada-jp/oinari-lib-go/internal/fox"
	"github.com/llamerada-jp/oinari-lib-go/oinari"
	"github.com/llamerada-jp/oinari-lib-go/wasm"
)

func main() {
	err := func() error {
		fox := &fox.Fox{}
		options := &oinari.ManagerOptions{
			Dialer: wasm.Dialer,
			Logger: wasm.NewLogger(),
		}
		manager, err := oinari.NewManagerWithRunnable3D(options, fox)
		if err != nil {
			return err
		}

		return manager.Start(context.Background())
	}()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
