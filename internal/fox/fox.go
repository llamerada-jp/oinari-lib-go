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
package fox

import (
	"context"
	_ "embed"
	"encoding/json"
	"math/rand"

	"github.com/llamerada-jp/oinari-lib-go/oinari"
)

var (
	//go:embed fox.glb
	model []byte
)

type Fox struct {
	oinari.UnimplementedRunnable3D
}

func (f *Fox) Start(ctx context.Context, op oinari.Operator3D) error {
	return op.SetModelGlTF(ctx, model)
}

func (f *Fox) Activate(ctx context.Context, op oinari.Operator3D, data map[string][]byte) error {
	sd := &Fox{}
	err := json.Unmarshal(data["fox"], sd)
	if err != nil {
		return err
	}

	return op.SetModelGlTF(ctx, model)
}

func (f *Fox) Step(ctx context.Context, op oinari.Operator3D) error {
	return op.Move(ctx, rand.Float64()*10.0-5.0, rand.Float64()*10.0-5.0, 0.0)
}

func (f *Fox) Dump(_ context.Context, op oinari.Operator3D) (map[string][]byte, error) {
	data, err := json.Marshal(Fox{})
	if err != nil {
		return nil, err
	}
	dataMap := make(map[string][]byte)
	dataMap["fox"] = data
	return dataMap, nil
}

func (f *Fox) Inactivate(_ context.Context, op oinari.Operator3D) error {
	return nil
}

func (f *Fox) Stop(_ context.Context, op oinari.Operator3D) error {
	return nil
}
