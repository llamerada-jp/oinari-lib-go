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
package wasm

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net"
	"regexp"
	"strconv"
	"sync"
	"syscall/js"
	"time"
)

type accepterEntry struct {
	serverPort uint16
	clientPort uint16
}

type connectorEntry struct {
	serverPort uint16
	clientPort uint16
	err        error
}

type listenerWasm struct {
	listenAddr *addrWasm
	accepter   chan *accepterEntry
}

type connWasm struct {
	localAddr  *addrWasm
	remoteAddr *addrWasm
	deadline   time.Time
	receiver   chan []byte
	buffer     []*[]byte
}

type addrWasm struct {
	port uint16
}

const (
	ACCEPTER_BUFFERS = 10
	RECEIVER_BUFFERS = 10
	JS_MODULE_NAME   = "net"
)

var (
	jsModule = js.Global().Get(JS_MODULE_NAME)

	accepters    = make(map[uint16]chan *accepterEntry)
	accepterMtx  sync.Mutex
	connectors   = make(map[uint32]chan *connectorEntry)
	connectorMtx sync.Mutex
	receivers    = make(map[uint16]chan []byte)
	receiverMtx  sync.Mutex
)

func init() {
	jsModule.Set("acceptor", js.FuncOf(acceptor))
	jsModule.Set("connectReply", js.FuncOf(connectReply))
	jsModule.Set("receiver", js.FuncOf(receiver))
	jsModule.Set("closer", js.FuncOf(closer))
}

func checkJsError(v js.Value) error {
	if v.IsNull() || v.IsUndefined() {
		return nil
	}

	errStr := v.String()
	if len(errStr) == 0 {
		panic("wasm module returns empty error message")
	}

	return fmt.Errorf(errStr)
}

/**
 * acceptor is receiver for wasm.
 * acceptor(listenPort, serverPort, clientPort uint16) string
 * js module must send connectReply with error if this method return error message.
 */
func acceptor(_ js.Value, args []js.Value) interface{} {
	listenPort := uint16(args[0].Int())
	serverPort := uint16(args[1].Int())
	clientPort := uint16(args[2].Int())

	accepterMtx.Lock()
	defer accepterMtx.Unlock()
	accepter, ok := accepters[listenPort]
	if !ok {
		return "the port is not on standby"
	}

	// Tell a new entry for Accept method.
	accepter <- &accepterEntry{
		serverPort: serverPort,
		clientPort: clientPort,
	}

	return nil
}

/**
 * connectReply is receiver for wasm.
 * connectReply(key uint32, serverPort, clientPort uint16, err string) string
 * js module must reset the connection if this method return error message.
 */
func connectReply(_ js.Value, args []js.Value) interface{} {
	key := uint32(args[0].Int())
	serverPort := uint16(args[1].Int())
	clientPort := uint16(args[2].Int())
	err := args[3]

	connectorMtx.Lock()
	connector, ok := connectors[key]
	connectorMtx.Unlock()
	if !ok {
		return "there is no connector waiting"
	}

	if err.IsNull() {
		connector <- &connectorEntry{
			serverPort: serverPort,
			clientPort: clientPort,
			err:        nil,
		}

	} else {
		connector <- &connectorEntry{
			serverPort: 0,
			clientPort: 0,
			err:        fmt.Errorf(err.String()),
		}
	}

	return nil
}

/**
 * receiver is receiver for wasm.
 * receiver(localPort uint16, data string) string
 * js module must reset the connection if this method return error message.
 */
func receiver(_ js.Value, args []js.Value) interface{} {
	targetPort := uint16(args[0].Int())
	data := args[1].String()

	receiverMtx.Lock()
	receiver, ok := receivers[targetPort]
	receiverMtx.Unlock()
	if !ok {
		return "target port is not open"
	}

	byteData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return err
	}
	receiver <- byteData

	return nil
}

/**
 * closer is receiver for wasm.
 * closer(targetPort uint16) nil
 */
func closer(_ js.Value, args []js.Value) interface{} {
	port := uint16(args[0].Int())

	receiverMtx.Lock()
	receiver, ok := receivers[port]
	receiverMtx.Unlock()
	if ok {
		close(receiver)
	}
	return nil
}

func Dialer(_ context.Context, target string) (net.Conn, error) {
	// Get targetPort from target string.
	exp := regexp.MustCompile(`^localhost:(\d+)$`)
	matched := exp.FindStringSubmatch(target)
	if len(matched) != 2 {
		return nil, fmt.Errorf("the target `%s` is unexpected format", target)
	}
	targetPort, _ := strconv.Atoi(matched[1])

	// Generate random unique key.
	connectorMtx.Lock()
	var key uint32
	for {
		key = rand.Uint32()
		if _, ok := connectors[key]; !ok {
			break
		}
	}

	connector := make(chan *connectorEntry)
	connectors[key] = connector
	connectorMtx.Unlock()

	if err := checkJsError(jsModule.Call("connect", key, uint16(targetPort))); err != nil {
		return nil, err
	}

	// Wait for getting result.
	resp := <-connector

	// Delete used connector
	connectorMtx.Lock()
	close(connector)
	delete(connectors, key)
	connectorMtx.Unlock()

	if resp.err != nil {
		return nil, resp.err
	}

	// Assign new receiver.
	receiver := make(chan []byte, RECEIVER_BUFFERS)
	receiverMtx.Lock()
	receivers[resp.clientPort] = receiver
	receiverMtx.Unlock()

	return &connWasm{
		localAddr:  newAddr(resp.clientPort),
		remoteAddr: newAddr(resp.serverPort),
		receiver:   receiver,
		buffer:     make([]*[]byte, 0),
	}, nil
}

//
func Listen(port uint16) (net.Listener, error) {
	accepterMtx.Lock()
	defer accepterMtx.Unlock()
	if _, ok := accepters[port]; ok {
		return nil, fmt.Errorf("port %d is already in use", port)
	}

	if err := checkJsError(jsModule.Call("listen", port)); err != nil {
		return nil, err
	}

	accepter := make(chan *accepterEntry)
	accepters[port] = accepter

	return &listenerWasm{
		listenAddr: newAddr(port),
		accepter:   accepter,
	}, nil
}

// Accept waits for and returns the next connection to the listener.
func (l *listenerWasm) Accept() (net.Conn, error) {
	entry, ok := <-l.accepter

	if !ok {
		return nil, fmt.Errorf("port %d has been closed", l.listenAddr.port)
	}

	// Check if the receiver exists.
	receiverMtx.Lock()
	defer receiverMtx.Unlock()
	if _, ok := receivers[entry.serverPort]; ok {
		panic("duplicate port")
	}

	receiver := make(chan []byte, RECEIVER_BUFFERS)
	receivers[entry.serverPort] = receiver

	return &connWasm{
		localAddr:  newAddr(entry.serverPort),
		remoteAddr: newAddr(entry.clientPort),
		receiver:   receiver,
	}, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *listenerWasm) Close() error {
	if err := checkJsError(jsModule.Call("close", l.listenAddr.port)); err != nil {
		return err
	}

	accepterMtx.Lock()
	defer accepterMtx.Unlock()

	close(l.accepter)
	delete(accepters, l.listenAddr.port)

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
		if len(c.buffer) != 0 {
			head := c.buffer[0]
			copyLen := copy(b, *head)
			if len(*head) == copyLen {
				c.buffer = c.buffer[1:]
			} else {
				*head = (*head)[copyLen:]
			}

			return copyLen, nil
		}

		if c.deadline.Before(time.Now()) {
			c.deadline = time.Time{}
		}
		var timer *time.Timer
		var ch <-chan time.Time
		if !c.deadline.IsZero() {
			timer = time.NewTimer(c.deadline.Sub(time.Now()))
			ch = timer.C
		} else {
			ch = make(chan time.Time)
		}
		select {
		case data, ok := <-c.receiver:
			if !ok {
				return 0, io.EOF
			}
			if data == nil {
				continue
			}
			c.buffer = append(c.buffer, &data)
			c.deadline = time.Time{}

		case <-ch:
			return 0, fmt.Errorf("timeout")
		}
	}
}

// Write writes data to the connection.
// Write can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetWriteDeadline.
func (c *connWasm) Write(b []byte) (int, error) {
	if err := checkJsError(jsModule.Call("write", c.localAddr.port, base64.StdEncoding.EncodeToString(b))); err != nil {
		return 0, err
	}
	return len(b), nil
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *connWasm) Close() error {
	if err := checkJsError(jsModule.Call("close", c.localAddr.port)); err != nil {
		return err
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

func newAddr(port uint16) *addrWasm {
	return &addrWasm{
		port: port,
	}
}

// name of the network (for example, "tcp", "udp")
func (a *addrWasm) Network() string {
	return "wasm"
}

// string form of address (for example, "192.0.2.1:25", "[2001:db8::1]:80")
func (a *addrWasm) String() string {
	return fmt.Sprintf("localhost:%d", a.port)
}
