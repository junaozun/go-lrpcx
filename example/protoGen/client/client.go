package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lubanproj/gorpc/client"

	pb "github.com/lubanproj/gorpc/examples/helloworld2/helloworld"
)

func main() {
	opts := []client.ClientOption{
		client.WithTarget("127.0.0.1:8000"),
		client.WithNetwork("tcp"),
		client.WithTimeout(2000 * time.Millisecond),
	}
	proxy := pb.NewGreeterClientProxy(opts...)
	req := &pb.HelloRequest{
		Msg: "hello",
	}
	rsp, err := proxy.SayHello(context.Background(), req, opts...)
	fmt.Println(rsp, err)
}
