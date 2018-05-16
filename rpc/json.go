// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/json"
	"errors"
	"io"
	"net/rpc"
	"sync"
)

const (
	jsonrpcVersion = "2.0"
)

type jsonCodec struct {
	dec *json.Decoder // for reading JSON values
	enc *json.Encoder // for writing JSON values
	c   io.Closer
	srv *rpc.Server

	// temporary work space
	req jsonRequest

	// JSON-RPC clients can use arbitrary json values as request IDs.
	// Package rpc expects uint64 request IDs.
	// We assign uint64 sequence numbers to incoming requests
	// but save the original request ID in the pending map.
	// When rpc responds, we use the sequence number in
	// the response to find the original request ID.
	mutex    sync.Mutex // protects seq, pending
	encmutex sync.Mutex // protects enc
	seq      uint64
	pending  map[uint64]*json.RawMessage
}

// NewJSONCodec returns a new rpc.ServerCodec using JSON-RPC on conn.
func NewJSONCodec(conn io.ReadWriteCloser, srv *rpc.Server) rpc.ServerCodec {
	if srv == nil {
		srv = rpc.DefaultServer
	}
	srv.Register(JSONRPC2{})
	return &jsonCodec{
		dec:     json.NewDecoder(conn),
		enc:     json.NewEncoder(conn),
		c:       conn,
		srv:     srv,
		pending: make(map[uint64]*json.RawMessage),
	}
}

type jsonRequest struct {
	Version string           `json:"jsonrpc"`
	Method  string           `json:"method"`
	Params  *json.RawMessage `json:"params"`
	ID      *json.RawMessage `json:"id"`
}

func (r *jsonRequest) UnmarshalJSON(raw []byte) error {
	r.reset()
	type req *jsonRequest
	if err := json.Unmarshal(raw, req(r)); err != nil {
		return errors.New("bad request")
	}

	var reqMap = make(map[string]*json.RawMessage)
	if err := json.Unmarshal(raw, &reqMap); err != nil {
		return errors.New("bad request")
	}
	if reqMap["jsonrpc"] == nil || reqMap["method"] == nil {
		return errors.New("bad request")
	}
	_, okID := reqMap["id"]
	_, okParams := reqMap["params"]
	if len(reqMap) == 3 && !(okID || okParams) || len(reqMap) == 4 && !(okID && okParams) || len(reqMap) > 4 {
		return errors.New("bad request")
	}
	if r.Version != "2.0" {
		return errors.New("bad request")
	}
	if okParams {
		if r.Params == nil || len(*r.Params) == 0 {
			return errors.New("bad request")
		}
	}
	if okID && r.ID == nil {
		r.ID = nil
	}
	if okID {
		if len(*r.ID) == 0 {
			return errors.New("bad request")
		}
		switch []byte(*r.ID)[0] {
		case 't', 'f', '{', '[':
			return errors.New("bad request")
		}
	}

	return nil
}

func (r *jsonRequest) reset() {
	r.Version = ""
	r.Method = ""
	r.Params = nil
	r.ID = nil
}

type jsonResponse struct {
	Version string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Result  interface{}      `json:"result,omitempty"`
	Error   interface{}      `json:"error,omitempty"`
}

func (c *jsonCodec) ReadRequestHeader(r *rpc.Request) error {
	var raw json.RawMessage
	if err := c.dec.Decode(&raw); err != nil {
		c.encmutex.Lock()
		c.enc.Encode(jsonResponse{Version: jsonrpcVersion, ID: &null, Error: errParse})
		c.encmutex.Unlock()
		return err
	}

	if len(raw) > 0 && raw[0] == '[' {
		c.req.Version = jsonrpcVersion
		c.req.Method = "JSONRPC2.Batch"
		c.req.Params = &raw
		c.req.ID = &null
	} else if err := json.Unmarshal(raw, &c.req); err != nil {
		if err.Error() == "bad request" {
			c.encmutex.Lock()
			c.enc.Encode(jsonResponse{Version: jsonrpcVersion, ID: &null, Error: errRequest})
			c.encmutex.Unlock()
		}
		return err
	}

	r.ServiceMethod = c.req.Method

	// JSON request id can be any JSON value;
	// RPC package expects uint64.  Translate to
	// internal uint64 and save JSON on the side.
	c.mutex.Lock()
	c.seq++
	c.pending[c.seq] = c.req.ID
	c.req.ID = nil
	r.Seq = c.seq
	c.mutex.Unlock()

	return nil
}

func (c *jsonCodec) ReadRequestBody(x interface{}) error {
	if x == nil {
		return nil
	}

	if c.req.Params == nil {
		return errParams
	}
	// JSON params is array value.
	// RPC params is struct.
	// Unmarshal into array containing struct for now.
	// Should think about making RPC more general.
	var params [1]interface{}
	params[0] = x

	if c.req.Method == "JSONRPC2.Batch" {
		arg := x.(*BatchArg)
		arg.srv = c.srv
		if err := json.Unmarshal(*c.req.Params, &arg.reqs); err != nil {
			return NewError(errParams.Code, err.Error())
		}
		if len(arg.reqs) == 0 {
			return errRequest
		}
	} else if err := json.Unmarshal(*c.req.Params, &params); err != nil {
		return NewError(errParams.Code, err.Error())
	}

	return nil
}

var null = json.RawMessage([]byte("null"))

func (c *jsonCodec) WriteResponse(r *rpc.Response, x interface{}) error {
	c.mutex.Lock()
	b, ok := c.pending[r.Seq]
	if !ok {
		c.mutex.Unlock()
		return errors.New("invalid sequence number in response")
	}
	delete(c.pending, r.Seq)
	c.mutex.Unlock()

	if replies, ok := x.(*[]*json.RawMessage); r.ServiceMethod == "JSONRPC2.Batch" && ok {
		if len(*replies) == 0 {
			return nil
		}
		c.encmutex.Lock()
		defer c.encmutex.Unlock()
		return c.enc.Encode(replies)
	}

	if b == nil {
		// Invalid request so no id. Use JSON null.
		b = &null
	}
	resp := jsonResponse{Version: jsonrpcVersion, ID: b}
	if r.Error == "" {
		if x == nil {
			resp.Result = &null
		} else {
			resp.Result = x
		}
	} else if r.Error[0] == '{' && r.Error[len(r.Error)-1] == '}' {
		raw := json.RawMessage(r.Error)
		resp.Error = &raw
	} else {
		raw := json.RawMessage(newError(r.Error).Error())
		resp.Error = &raw
	}
	c.encmutex.Lock()
	defer c.encmutex.Unlock()
	return c.enc.Encode(resp)
}

func (c *jsonCodec) Close() error {
	return c.c.Close()
}

// ServeConn runs the JSON-RPC server on a single connection.
// ServeConn blocks, serving the connection until the client hangs up.
// The caller typically invokes ServeConn with go-routine.
func ServeConn(conn io.ReadWriteCloser) {
	rpc.ServeCodec(NewJSONCodec(conn, nil))
}
