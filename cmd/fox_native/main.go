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
	"flag"
	"fmt"
	"os"

	"github.com/llamerada-jp/oinari-lib-go/pkg/fox"
	"github.com/llamerada-jp/oinari-lib-go/pkg/oinari"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	hubAddress string
)

var cmd = &cobra.Command{
	Use: "fox",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		flag.CommandLine.Parse([]string{})
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		fox := &fox.Fox{}
		options := &oinari.ManagerOptions{
			HubAddress: hubAddress,
		}
		manager, err := oinari.NewManagerWithRunnable3D(options, fox)
		if err != nil {
			return err
		}

		return manager.Start(context.Background())
	},
}

func init() {
	flags := cmd.PersistentFlags()

	flags.StringVarP(&hubAddress, "address", "a", "localhost:1984", "the address hub program waiting")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
}

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
