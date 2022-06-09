package main

import (
	"context"
	"fmt"
	"github.com/junaozun/go-lrpxc"
	"time"

	"github.com/lubanproj/gorpc/examples/helloworld2/helloworld"
)

type greeterService struct{}

func (g *greeterService) SayHello(ctx context.Context, req *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	fmt.Println("recv Msg : ", req.Msg)
	rsp := &helloworld.HelloReply{
		Msg: req.Msg + " world",
	}
	return rsp, nil
}

func main() {
	opts := []github.com / junaozun /
	go -lrpxc.ServerOption{
		github.com / junaozun /go -lrpxc.WithAddress("127.0.0.1:8000"),
		github.com/junaozun/go -lrpxc.WithNetwork("tcp"),
		github.com/junaozun/go -lrpxc.WithProtocol("proto"),
		github.com/junaozun/go -lrpxc.WithTimeout(time.Millisecond * 2000),
	}
	s := github.com/junaozun/go-lrpxc.NewServer(opts...)
	helloworld.RegisterService(s, &greeterService{})
	s.Serve()
}
