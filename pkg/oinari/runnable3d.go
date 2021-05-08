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
package oinari

import (
	"context"
	"fmt"

	"github.com/llamerada-jp/oinari-lib-go/pkg/api"
)

type Runnable3D interface {
	Start(context.Context, Operator3D) error
	Activate(context.Context, Operator3D, map[string][]byte) error
	Step(context.Context, Operator3D) error
	Dump(context.Context, Operator3D) (map[string][]byte, error)
	Inactivate(context.Context, Operator3D) error
	Stop(context.Context, Operator3D) error
}

type Operator3D interface {
	GetAbsolutePosition(context.Context) (float64, float64, float64, error)
	SetAbsolutePosition(context.Context, float64, float64, float64) error
	Move(context.Context, float64, float64, float64) error

	SetModelGlTF(context.Context, []byte) error
}

type runnable3DImpl struct {
	operator *operator3DImpl
	runnable Runnable3D
}

func newRunnableWithRunnable3D(runnable Runnable3D, client api.OinariClient) runnable {
	return &runnable3DImpl{
		operator: &operator3DImpl{
			client: client,
		},
		runnable: runnable,
	}
}

func (impl *runnable3DImpl) Start(ctx context.Context) error {
	impl.operator.loadInitParameters(ctx)
	return impl.runnable.Start(ctx, impl.operator)
}

func (impl *runnable3DImpl) Activate(ctx context.Context, data map[string][]byte) error {
	impl.operator.loadInitParameters(ctx)
	return impl.runnable.Activate(ctx, impl.operator, data)
}

func (impl *runnable3DImpl) Step(ctx context.Context) error {
	return impl.runnable.Step(ctx, impl.operator)
}

func (impl *runnable3DImpl) Dump(ctx context.Context) (map[string][]byte, error) {
	return impl.runnable.Dump(ctx, impl.operator)
}

func (impl *runnable3DImpl) Inactivate(ctx context.Context) error {
	return impl.runnable.Inactivate(ctx, impl.operator)
}

func (impl *runnable3DImpl) Stop(ctx context.Context) error {
	return impl.runnable.Stop(ctx, impl.operator)
}

type operator3DImpl struct {
	client api.OinariClient
	// cache position
	x float64
	y float64
	z float64
}

func (op *operator3DImpl) GetAbsolutePosition(ctx context.Context) (float64, float64, float64, error) {
	req := api.GetPositionRequest{
		Type: api.CoordinateType_COORDINATE_3D,
	}
	res, err := op.client.GetPosition(ctx, &req)
	if err != nil {
		return 0, 0, 0, err
	}

	coordinate := res.GetCoordinate()
	if coordinate.GetType() != api.CoordinateType_COORDINATE_3D {
		return 0, 0, 0, fmt.Errorf("unexpected coordinate type: %d", coordinate.GetType())
	}

	d3 := coordinate.GetD3()
	op.x = d3.GetX()
	op.y = d3.GetY()
	op.z = d3.GetZ()
	return op.x, op.y, op.z, nil
}

func (op *operator3DImpl) SetAbsolutePosition(ctx context.Context, x float64, y float64, z float64) error {
	req := api.SetPositionRequest{
		Coordinate: &api.Coordinate{
			Type: api.CoordinateType_COORDINATE_3D,
			Coordinate: &api.Coordinate_D3_{
				D3: &api.Coordinate_D3{
					X: x,
					Y: y,
					Z: z,
				},
			},
		},
	}
	_, err := op.client.SetPosition(ctx, &req)
	if err != nil {
		return err
	}

	op.x = x
	op.y = y
	op.z = z
	return nil
}

func (op *operator3DImpl) Move(ctx context.Context, x float64, y float64, z float64) error {
	req := api.SetPositionRequest{
		Coordinate: &api.Coordinate{
			Type: api.CoordinateType_COORDINATE_3D,
			Coordinate: &api.Coordinate_D3_{
				D3: &api.Coordinate_D3{
					X: op.x + x,
					Y: op.y + y,
					Z: op.z + z,
				},
			},
		},
	}
	_, err := op.client.SetPosition(ctx, &req)
	if err != nil {
		return err
	}

	op.x += x
	op.y += y
	op.z += z
	return nil
}

func (op *operator3DImpl) SetModelGlTF(ctx context.Context, data []byte) error {
	req := api.SetModelRequest{
		Type: api.SetModelRequest_GLTF,
		Data: data,
	}
	_, err := op.client.SetModel(ctx, &req)
	return err
}

func (op *operator3DImpl) loadInitParameters(ctx context.Context) error {
	req := api.GetPositionRequest{
		Type: api.CoordinateType_COORDINATE_3D,
	}
	res, err := op.client.GetPosition(ctx, &req)
	if err != nil {
		return err
	}

	coordinate := res.GetCoordinate()
	if coordinate.GetType() != api.CoordinateType_COORDINATE_3D {
		return fmt.Errorf("unexpected coordinate type: %d", coordinate.GetType())
	}

	d3 := coordinate.GetD3()
	op.x = d3.GetX()
	op.y = d3.GetY()
	op.z = d3.GetZ()
	return nil
}
