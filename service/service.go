package service

import (
	"go/ast"
	"log"
	"reflect"
)

type service struct {
	name   string
	typ    reflect.Type
	rcvr   reflect.Value
	method map[string]*methodType
}

func NewService(rcvr interface{}) *service {
	s := new(service)
	s.rcvr = reflect.ValueOf(rcvr)
	s.name = reflect.Indirect(s.rcvr).Type().Name()
	s.typ = reflect.TypeOf(rcvr)
	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.name)
	}
	s.registerMethods()
	return s
}

func (s *service) registerMethods() {

}

func isExportedOrBuiltinType(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}
