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
	"io"
	"net"
	"time"

	"github.com/llamerada-jp/oinari-lib-go/api"
	"github.com/llamerada-jp/oinari-lib-go/logger"
	"google.golang.org/grpc"
)

type Manager interface {
	Start(context.Context) error
}

type ManagerOptions struct {
	Dialer     func(context.Context, string) (net.Conn, error)
	HubAddress string
	Logger     logger.Logger
}

type managerImpl struct {
	options  *ManagerOptions
	log      logger.Log
	conn     *grpc.ClientConn
	client   api.OinariClient
	runnable runnable
}

func NewManagerWithRunnable3D(options *ManagerOptions, runnable Runnable3D) (Manager, error) {
	lgr := options.Logger
	if lgr == nil {
		lgr = logger.LoggerWithStdLog()
	}
	log := logger.NewLog(lgr)

	dialOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: 10 * time.Second,
		}),
	}
	if options.Dialer != nil {
		dialOpts = append(dialOpts, grpc.WithContextDialer(options.Dialer))
	}

	target := options.HubAddress
	if len(target) == 0 {
		target = "localhost:1984"
	}

	conn, err := grpc.Dial(
		target,
		dialOpts...,
	)
	if err != nil {
		return nil, err
	}

	client := api.NewOinariClient(conn)

	return &managerImpl{
		options:  options,
		log:      log,
		conn:     conn,
		client:   client,
		runnable: newRunnableWithRunnable3D(runnable, client),
	}, nil
}

func (mi *managerImpl) Start(ctx context.Context) error {
	err := mi.start(ctx)
	if err != nil {
		return err
	}

	err = mi.loop(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (mi *managerImpl) start(ctx context.Context) error {
	req := api.GetApplicationInformationRequest{}
	res, err := mi.client.GetApplicationInformation(ctx, &req)
	if err != nil {
		return err
	}
	switch res.GetStatus() {
	case api.GetApplicationInformationResponse_STATUS_STARTING:
		return mi.runnable.Start(ctx)

	case api.GetApplicationInformationResponse_STATUS_MIGRATING:
		req := api.GetDumpRequest{}
		res, err := mi.client.GetDump(ctx, &req)
		if err != nil {
			return err
		}
		return mi.runnable.Activate(ctx, res.GetData())

	default:
		return fmt.Errorf("unexpected application status %d", res.GetStatus())
	}
}

func (mi *managerImpl) loop(ctx context.Context) error {
	defer mi.conn.Close()

	req := api.GetNodeEventRequest{}
	stream, err := mi.client.GetNodeEvent(ctx, &req)
	if err != nil {
		return err
	}

	for {
		res, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		switch res.GetType() {
		case api.GetNodeEventResponse_EVENT_STEP:
			err = mi.runnable.Step(ctx)
			if err != nil {
				return err
			}

		case api.GetNodeEventResponse_EVENT_STOP:
			return mi.runnable.Stop(ctx)

		case api.GetNodeEventResponse_EVENT_INACTIVATE:
			data, err := mi.runnable.Dump(ctx)
			if err != nil {
				return err
			}

			req := api.PutDumpRequest{
				Data: data,
			}
			_, err = mi.client.PutDump(ctx, &req)
			if err != nil {
				return err
			}

			err = mi.runnable.Inactivate(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
