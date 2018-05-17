// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/rpc"
	"strings"
	"testing"
)

type Args struct {
	A, B int
}

type Reply struct {
	C int
}

type Arith int

type ArithAddResp struct {
	ID     interface{} `json:"id"`
	Result Reply       `json:"result"`
	Error  interface{} `json:"error"`
}

func (t *Arith) Add(args *Args, reply *Reply) error {
	reply.C = args.A + args.B
	return nil
}

func (t *Arith) Mul(args *Args, reply *Reply) error {
	reply.C = args.A * args.B
	return nil
}

func (t *Arith) Div(args *Args, reply *Reply) error {
	if args.B == 0 {
		return errors.New("divide by zero")
	}
	reply.C = args.A / args.B
	return nil
}

func (t *Arith) Error(args *Args, reply *Reply) error {
	panic("ERROR")
}

type BuiltinTypes struct{}

func (BuiltinTypes) Map(i int, reply *map[int]int) error {
	(*reply)[i] = i
	return nil
}

func (BuiltinTypes) Slice(i int, reply *[]int) error {
	*reply = append(*reply, i)
	return nil
}

func (BuiltinTypes) Array(i int, reply *[1]int) error {
	(*reply)[0] = i
	return nil
}

func init() {
	rpc.Register(new(Arith))
	rpc.Register(BuiltinTypes{})
}

func Test_ServerNoParams(t *testing.T) {
	cli, srv := net.Pipe()
	defer cli.Close()
	go ServeConn(srv)
	dec := json.NewDecoder(cli)

	fmt.Fprintf(cli, `{"jsonrpc":"2.0" , "method": "Arith.Add", "id": "123"}`)
	var resp ArithAddResp
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("Decode after no params: %s", err)
	}
	if resp.Error == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func Test_ServerEmptyMessage(t *testing.T) {
	cli, srv := net.Pipe()
	defer cli.Close()
	go ServeConn(srv)
	dec := json.NewDecoder(cli)

	fmt.Fprintf(cli, "{}")
	var resp ArithAddResp
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("Decode after empty: %s", err)
	}
	if resp.Error == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func Test_Server(t *testing.T) {
	cli, srv := net.Pipe()
	defer cli.Close()
	go ServeConn(srv)
	dec := json.NewDecoder(cli)

	// Send hand-coded requests to server, parse responses.
	for i := 0; i < 10; i++ {
		fmt.Fprintf(cli, `{"jsonrpc": "2.0" ,"method": "Arith.Add", "id": "\u%04d", "params": [{"A": %d, "B": %d}]}`, i, i, i+1)
		var resp ArithAddResp
		err := dec.Decode(&resp)
		if err != nil {
			t.Fatalf("Decode: %s", err)
		}
		if resp.Error != nil {
			t.Fatalf("resp.Error: %s", resp.Error)
		}
		if resp.ID.(string) != string(i) {
			t.Fatalf("resp: bad id %q want %q", resp.ID.(string), string(i))
		}
		if resp.Result.C != 2*i+1 {
			t.Fatalf("resp: bad result: %d+%d=%d", i, i+1, resp.Result.C)
		}
	}
}

func Test_MalformedInput(t *testing.T) {
	cli, srv := net.Pipe()
	go func() {
		cli.Write([]byte(`{id:1}`)) // invalid
		cli.Read(make([]byte, 1024))
	}()
	ServeConn(srv) // must return, not loop
}

func Test_Json_ErrorHasNullResult(t *testing.T) {
	var out bytes.Buffer
	sc := NewJSONCodec(struct {
		io.Reader
		io.Writer
		io.Closer
	}{
		Reader: strings.NewReader(`{"jsonrpc":"2.0", "method": "Arith.Add", "id": "123", "params": []}`),
		Writer: &out,
		Closer: ioutil.NopCloser(nil),
	}, nil)
	r := new(rpc.Request)
	if err := sc.ReadRequestHeader(r); err != nil {
		t.Fatal(err)
	}
	const valueText = "the value we don't want to see"
	const errorText = "some error"
	err := sc.WriteResponse(&rpc.Response{
		ServiceMethod: "Method",
		Seq:           1,
		Error:         errorText,
	}, valueText)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), errorText) {
		t.Fatalf("Response didn't contain expected error %q: %s", errorText, &out)
	}
	if strings.Contains(out.String(), valueText) {
		t.Errorf("Response contains both an error and value: %s", &out)
	}
}

func Test_UnexpectedError(t *testing.T) {
	cli, srv := myPipe()
	go func() {
		cli.PipeWriter.CloseWithError(errors.New("unexpected error")) // reader will get this error
		cli.PipeReader.Close()                                        // writer will get ErrClosedPipe
	}()
	ServeConn(srv) // must return, not loop

}

// Copied from package net.
func myPipe() (*pipe, *pipe) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	return &pipe{r1, w2}, &pipe{r2, w1}
}

type pipe struct {
	*io.PipeReader
	*io.PipeWriter
}

type pipeAddr int

func (pipeAddr) Network() string {
	return "pipe"
}

func (pipeAddr) String() string {
	return "pipe"
}

func (p *pipe) Close() error {
	err := p.PipeReader.Close()
	err1 := p.PipeWriter.Close()
	if err == nil {
		err = err1
	}
	return err
}

func (p *pipe) LocalAddr() net.Addr {
	return pipeAddr(0)
}

func (p *pipe) RemoteAddr() net.Addr {
	return pipeAddr(0)
}

func (p *pipe) SetTimeout(nsec int64) error {
	return errors.New("net.Pipe does not support timeouts")
}

func (p *pipe) SetReadTimeout(nsec int64) error {
	return errors.New("net.Pipe does not support timeouts")
}

func (p *pipe) SetWriteTimeout(nsec int64) error {
	return errors.New("net.Pipe does not support timeouts")
}
