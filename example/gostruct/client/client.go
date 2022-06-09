package main

import (
	"context"
	"fmt"
	"time"

	"github.com/junaozun/go-lrpxc/example/gostruct/way"

	"github.com/lubanproj/gorpc/client"
)

func main() {

	opts := []client.ClientOption{
		client.WithTarget("127.0.0.1:8000"),
		client.WithNetwork("tcp"),
		client.WithTimeout(2000 * time.Millisecond),
		client.WithSerializationType("msgpack"),
	}
	c := client.DefaultClient
	req := &way.HelloRequest{
		Msg: "hello",
	}
	rsp := &way.HelloReply{}
	err := c.Call(context.Background(), "/helloworld.Greeter/SayHello", req, rsp, opts...)
	fmt.Println(rsp.Msg, err)
}
