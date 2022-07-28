package service

import (
	"errors"
	"sync"
)

type server struct {
	serviceMap sync.Map
}

var DefaultServer = server{}

func Register(rcvr interface{}) error {
	return DefaultServer.Register(rcvr)
}

func (server *server) Register(rcvr interface{}) error {
	s := NewService(rcvr)
	if _, dup := server.serviceMap.LoadOrStore(s.name, s); dup {
		return errors.New("rpc: service already defined: " + s.name)
	}
	return nil
}

//func (server *server) findService(serviceMethod string)
