package client

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ylt94/mrpc/core"
	"github.com/ylt94/mrpc/mrpc"
)

type Call struct {
	Seq           uint64
	ServiceMethod string
	Args          interface{}
	Reply         interface{}
	Error         error
	Done          chan *Call
}

func (call *Call) done() {
	call.Done <- call
}

type Client struct {
	cc       core.Codec
	opt      *mrpc.Option
	sending  sync.Mutex
	header   core.Header
	mu       sync.Mutex
	seq      uint64
	pending  map[uint64]*Call
	closing  bool
	shutdown bool
}

var _ io.Closer = (*Client)(nil)

var ErrShutdown = errors.New("connection is shut down")

func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.closing {
		return ErrShutdown
	}

	client.closing = true
	return client.cc.Close()
}

func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()

	return !client.shutdown && !client.closing
}

func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.closing || client.shutdown {
		return 0, ErrShutdown
	}

	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++
	return call.Seq, nil
}

func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()

	call := client.pending[seq]
	delete(client.pending, seq)

	return call
}

func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()

	client.mu.Lock()
	defer client.mu.Unlock()

	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}

func (client *Client) recevie() {
	var err error
	for err == nil {
		var h core.Header
		//call 存在，但服务端处理出错，即 h.Error 不为空
		if err = client.cc.ReadHeader(&h); err != nil {
			break
		}
		call := client.removeCall(h.Seq)
		switch {
		case call == nil:
			err = client.cc.ReadBody(nil)
		case h.Error != "":
			call.Error = fmt.Errorf(h.Error)
			err = client.cc.ReadBody(nil)
			call.done()
		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}

	// error occurs, so terminateCalls pending calls
	client.terminateCalls(err)
}
