package mrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/ylt94/mrpc/core"
)

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int
	CodecType   core.Type
	ConnectTimeout time.Duration //0表示不限制
	HandleTileout time.Duration
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   core.GobType,
	ConnectTimeout: time.Second*5,
}

type Server struct {
	serviceMap sync.Map
}

func NewServer() *Server {
	return &Server{}
}

func Accept(lis net.Listener) { DefaultServer.Accept(lis) }

var DefaultServer = NewServer()

func Register(rcvr interface{}) error {
	return DefaultServer.Register(rcvr)
}

func (server *Server) Register(rcvr interface{}) error {
	s := NewService(rcvr)
	if _, dup := server.serviceMap.LoadOrStore(s.name, s); dup {
		return errors.New("rpc: service already defined: " + s.name)
	}
	return nil
}

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
	mtype        *methodType
	svc          *service
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
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}

	req := &request{h: h}
	req.svc, req.mtype, err = server.findService(h.ServiceMethod)
	if err != nil {
		return nil, err
	}
	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()

	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}
	if err = cc.ReadBody(argvi); err != nil {
		log.Println("rpc server: read body err:", err)
		return req, err
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

func (server *Server) handleRequest(cc core.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()
	//处理完
	called := make(chan struct{})
	//结果已发送
	sent := make(chan struct{})
	go func() {
		err := req.svc.call(req.mtype, req.argv, req.replyv)
		called <- struct{}{}
		if err != nil {
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			sent <- struct{}{}
			return
		}
	}()

	if timeout == 0 {
		<- called
		<- sent
		return
	}

	select {
		case <-time.After(timeout) :
			req.h.Error = fmt.Sprintf("rpc server: request handle timeout: expect within %s", timeout)
			server.sendResponse(cc, req.h, invalidRequest, sending)
		case <- called:
			<- sent
	}
}

func (server *Server) findService(serviceMethod string) (svc *service, mtype *methodType, err error) {
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc server: service/method request ill-formed: " + serviceMethod)
		return
	}
	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]
	svci, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server: can't find service " + serviceName)
		return
	}
	svc = svci.(*service)
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc server: can't find method " + methodName)
	}
	return
}
