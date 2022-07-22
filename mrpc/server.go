package mrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"

	"github.com/ylt94/mrpc/core"
)

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int
	CodecType   core.Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   core.GobType,
}

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

func Accept(lis net.Listener) { DefaultServer.Accept(lis) }

var DefaultServer = NewServer()

func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		//开启协程处理链接
		go server.ServerConn(conn)
	}
}

func (server *Server) ServerConn(conn io.ReadWriteCloser) {
	defer func() {
		_ = conn.Close()
	}()

	//TODO
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options json decode error", err)
		return
	}

	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}

	//获取对应协议类实例化函数
	f := core.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}

	//实例化协议类并开始处理
	server.serverCodec(f(conn))
}

var invalidRequest = struct{}{}

func (server *Server) serverCodec(cc core.Codec) {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)

	for {
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break
			}
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
		}
		wg.Add(1)
		//处理请求，应答
		go server.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

type request struct {
	h            *core.Header
	argv, replyv reflect.Value
}

func (server *Server) readRequestHeader(cc core.Codec) (*core.Header, error) {
	//TODO
	var h core.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) readRequest(cc core.Codec) (*request, error) {
	//获取header内容
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}

	//获取body内容 //TODO
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
		return nil, err
	}
	return req, nil
}

func (server *Server) sendResponse(cc core.Codec, h *core.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

func (server *Server) handleRequest(cc core.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	// TODO, should call registered rpc methods to get the right replyv
	// day 1, just print argv and send a hello message
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	//TODO
	req.replyv = reflect.ValueOf(fmt.Sprintf("mrpc resp %d", req.h.Seq))
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}
