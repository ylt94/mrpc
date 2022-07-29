package main

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/ylt94/mrpc/client"
	"github.com/ylt94/mrpc/mrpc"
)

// func startServer(addr chan string) {
// 	lis, err := net.Listen("tcp", ":8080")
// 	if err != nil {
// 		log.Fatal("server network error", err)
// 	}

// 	log.Println("server start listen:", lis.Addr())

// 	addr <- lis.Addr().String()
// 	mrpc.Accept(lis)
// }

func startServer(addr chan string) {
	var foo Foo
	if err := mrpc.Register(&foo); err != nil {
		log.Fatal("register error:", err)
	}
	// pick a free port
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	mrpc.Accept(l)
}

func main() {
	//DAY 1
	// addr := make(chan string)
	// go startServer(addr)
	// time.Sleep(time.Second*8)
	// conn, err := net.Dial("tcp", <-addr)
	// if err != nil {
	// 	log.Fatal("client dial err", err)
	// }
	// defer func() {
	// 	_ = conn.Close()
	// }()
	// log.Println("client dial succ")
	// time.Sleep(time.Second)

	// err = json.NewEncoder(conn).Encode(mrpc.DefaultOption)
	// if err != nil {
	// 	log.Fatal("json decode err", err)
	// }

	// cc := core.NewGobCodec(conn)
	// for i := 1; i <= 5; i++ {
	// 	h := &core.Header{
	// 		ServiceMethod: "Foo.Sum",
	// 		Seq:           uint64(i),
	// 	}

	// 	_ = cc.Write(h, fmt.Sprintf("client send requset %d", h.Seq))
	// 	_ = cc.ReadHeader(h)

	// 	var resp string
	// 	_ = cc.ReadBody(&resp)
	// 	log.Println("client get response:", resp)
	// }

	//DAY 2
	// log.SetFlags(0)
	// addr := make(chan string)
	// go startServer(addr)

	// client, err := client.Dail("tcp", <-addr)
	// if err != nil {
	// 	log.Fatal("client create err:", err)
	// }
	// defer func() { _ = client.Close() }()

	// time.Sleep(time.Second)
	// var wg sync.WaitGroup

	// for i := 0; i < 5; i++ {
	// 	wg.Add(1)
	// 	go func(i int) {
	// 		defer wg.Done()
	// 		args := fmt.Sprintf("mrpc req %d", i)
	// 		var reply string
	// 		if err := client.Call("Test.test", args, &reply); err != nil {
	// 			log.Fatal("client call Test.test error:", err)
	// 		}
	// 		log.Println("reply:", reply)
	// 	}(i)
	// }
	// wg.Wait()

	//DAY 3
	log.SetFlags(0)
	addr := make(chan string)
	go startServer(addr)
	client, _ := client.Dial("tcp", <-addr)
	defer func() { _ = client.Close() }()

	time.Sleep(time.Second)
	// send request & receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := &Args{Num1: i, Num2: i * i}
			var reply int
			if err := client.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)
		}(i)
	}
	wg.Wait()

}

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}
