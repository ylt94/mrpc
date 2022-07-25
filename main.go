package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/ylt94/mrpc/core"

	"github.com/ylt94/mrpc/mrpc"
)

func startServer(addr chan string) {
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("server network error", err)
	}

	log.Println("server start listen:", lis.Addr())

	addr <- lis.Addr().String()
	mrpc.Accept(lis)
}

func main() {
	addr := make(chan string)
	go startServer(addr)
	time.Sleep(time.Second*8)
	conn, err := net.Dial("tcp", <-addr)
	if err != nil {
		log.Fatal("client dial err", err)
	}
	defer func() {
		_ = conn.Close()
	}()
	log.Println("client dial succ")
	time.Sleep(time.Second)

	err = json.NewEncoder(conn).Encode(mrpc.DefaultOption)
	if err != nil {
		log.Fatal("json decode err", err)
	}

	cc := core.NewGobCodec(conn)
	for i := 1; i <= 5; i++ {
		h := &core.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}

		_ = cc.Write(h, fmt.Sprintf("client send requset %d", h.Seq))
		_ = cc.ReadHeader(h)

		var resp string
		_ = cc.ReadBody(&resp)
		log.Println("client get response:", resp)
	}
}
