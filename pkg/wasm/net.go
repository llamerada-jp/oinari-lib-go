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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"sync"
	"syscall/js"
	"time"
)

const (
	ACCEPTER_BUFFERS = 10
	RECEIVER_BUFFERS = 10
)

var (
	jsNet       = js.Global().Get("net")
	accepters   map[uint16]chan *accepterEntry
	accepterMtx sync.Mutex
	receivers   map[uint16]chan *string
	receiverMtx sync.Mutex
)

type accepterEntry struct {
	localPort  uint16
	remotePort uint16
}

type listenerWasm struct {
	listenAddr *addrWasm
	accepter   chan *accepterEntry
}

type connWasm struct {
	localAddr  *addrWasm
	remoteAddr *addrWasm
	deadline   time.Time
	receiver   chan *string
	buffer     []byte
}

type addrWasm struct {
	port uint16
}

func init() {
	// net.passDataCB(port uint16, data string) error
	jsNet.Set("passDataCB", js.FuncOf(jsPassData))
	// net.closeConnCB(port uint16)
	jsNet.Set("closeConnCB", js.FuncOf(jsCloseConn))
}

func jsPassData(_ js.Value, args []js.Value) interface{} {
	port := uint16(args[0].Int())
	data := args[1].String()

	receiver, ok := receivers[port]
	if !ok {
		return fmt.Errorf("port %d is not open", port)
	}
	receiver <- &data
	return nil
}

func jsCloseConn(_ js.Value, args []js.Value) interface{} {
	port := uint16(args[0].Int())
	receiver, ok := receivers[port]
	if ok {
		close(receiver)
		delete(receivers, port)
	}
	return nil
}

func Dialer(_ context.Context, target string) (net.Conn, error) {
	exp := regexp.MustCompile(`^localhost:(\d+)$`)
	matched := exp.FindStringSubmatch(target)
	if len(matched) != 2 {
		return nil, fmt.Errorf("the target `%s` is unexpected format", target)
	}
	port, _ := strconv.Atoi(matched[1])

	resp := struct {
		LocalPort  uint16 `json:"local,omitempty"`
		RemotePort uint16 `json:"remote,omitempty"`
		Error      string `json:"error,omitempty"`
	}{}

	// Call `net.connect(port)`.
	resStr := jsNet.Call("connect", uint16(port)).String()
	// Decode responce as JSON.
	if err := json.Unmarshal([]byte(resStr), &resp); err != nil {
		return nil, err
	}
	if len(resp.Error) != 0 {
		return nil, fmt.Errorf(resp.Error)
	}

	// Assign new receiver.
	receiverMtx.Lock()
	defer receiverMtx.Unlock()
	receivers[resp.LocalPort] = make(chan *string, RECEIVER_BUFFERS)

	return &connWasm{
		localAddr: &addrWasm{
			port: resp.LocalPort,
		},
		remoteAddr: &addrWasm{
			port: resp.RemotePort,
		},
		receiver: receivers[resp.LocalPort],
	}, nil
}

//
func Listen(port uint16) (net.Listener, error) {
	accepterMtx.Lock()
	defer accepterMtx.Unlock()

	if accepters == nil {
		accepters = make(map[uint16]chan *accepterEntry)
		// net.bindConn(listenPort, localPort, remotePort)
		jsNet.Set("bindConnCB", js.FuncOf(jsBindConn))
	}

	if _, ok := accepters[port]; ok {
		return nil, fmt.Errorf("port %d is already in use", port)
	}

	accepter := make(chan *accepterEntry)
	accepters[port] = accepter

	return &listenerWasm{
		listenAddr: &addrWasm{
			port: port,
		},
		accepter: accepter,
	}, nil
}

func jsBindConn(_ js.Value, args []js.Value) interface{} {
	listenPort := uint16(args[0].Int())
	localPort := uint16(args[1].Int())
	remotePort := uint16(args[2].Int())

	// Check if the port is listening.
	accepter, ok := accepters[listenPort]
	if !ok {
		return fmt.Errorf("port %d is not on standby", listenPort)
	}

	// Assign new receiver.
	receiverMtx.Lock()
	defer receiverMtx.Unlock()
	receivers[localPort] = make(chan *string, RECEIVER_BUFFERS)

	// Tell a new entry for Accept method.
	accepter <- &accepterEntry{
		localPort:  localPort,
		remotePort: remotePort,
	}

	return nil
}

// Accept waits for and returns the next connection to the listener.
func (l *listenerWasm) Accept() (net.Conn, error) {
	entry, ok := <-l.accepter

	if !ok {
		return nil, fmt.Errorf("port %d has been closed", l.listenAddr.port)
	}

	accepterMtx.Lock()
	defer accepterMtx.Unlock()

	// Check if the receiver exists.
	receiver, ok := receivers[entry.localPort]
	if !ok {
		return nil, fmt.Errorf("failed to accept")
	}

	return &connWasm{
		localAddr: &addrWasm{
			port: entry.localPort,
		},
		remoteAddr: &addrWasm{
			port: entry.remotePort,
		},
		receiver: receiver,
	}, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *listenerWasm) Close() error {
	accepterMtx.Lock()
	defer accepterMtx.Unlock()

	delete(accepters, l.listenAddr.port)
	close(l.accepter)

	return nil
}

// Addr returns the listener's network address.
func (l *listenerWasm) Addr() net.Addr {
	return l.listenAddr
}

// Read reads data from the connection.
// Read can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetReadDeadline.
func (c *connWasm) Read(b []byte) (n int, err error) {
	for {
		copyLen := copy(b, c.buffer)
		if copyLen != 0 {
			c.buffer = c.buffer[copyLen:]
			return copyLen, nil
		}

		if c.deadline.Before(time.Now()) {
			c.deadline = time.Time{}
		}
		var timer *time.Timer
		if !c.deadline.IsZero() {
			timer = time.NewTimer(c.deadline.Sub(time.Now()))
		}
		select {
		case data, ok := <-c.receiver:
			if !ok {
				return 0, io.EOF
			}
			if data == nil {
				continue
			}
			c.buffer = []byte(*data)
			c.deadline = time.Time{}

		case <-timer.C:
			return 0, fmt.Errorf("timeout")
		}
	}
}

// Write writes data to the connection.
// Write can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetWriteDeadline.
func (c *connWasm) Write(b []byte) (n int, err error) {
	resp := jsNet.Call("passData", string(b))
	if !resp.IsNull() {
		return 0, fmt.Errorf(resp.String())
	}
	return len(b), nil
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *connWasm) Close() error {
	resp := jsNet.Call("closeConn", c.localAddr)
	if !resp.IsNull() {
		return fmt.Errorf(resp.String())
	}
	return nil
}

// LocalAddr returns the local network address.
func (c *connWasm) LocalAddr() net.Addr {
	return c.localAddr
}

// RemoteAddr returns the remote network address.
func (c *connWasm) RemoteAddr() net.Addr {
	return c.remoteAddr
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
func (c *connWasm) SetDeadline(t time.Time) error {
	c.deadline = t
	c.receiver <- nil
	return nil
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *connWasm) SetReadDeadline(t time.Time) error {
	c.deadline = t
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// In this Conn implement, Write method keep the data in queue and it should not block and fail.
func (c *connWasm) SetWriteDeadline(t time.Time) error {
	return nil
}

// name of the network (for example, "tcp", "udp")
func (a *addrWasm) Network() string {
	return "wasm"
}

// string form of address (for example, "192.0.2.1:25", "[2001:db8::1]:80")
func (a *addrWasm) String() string {
	return fmt.Sprintf("localhost:%d", a.port)
}
