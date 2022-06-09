package main

import (
	"github.com/junaozun/go-lrpxc"
	"github.com/junaozun/go-lrpxc/example/gostruct/way"
	"time"
)

func main() {
	opts := []github.com / junaozun /
	go -lrpxc.ServerOption{
		github.com / junaozun /go -lrpxc.WithAddress("127.0.0.1:8000"),
		github.com/junaozun/go -lrpxc.WithNetwork("tcp"),
		github.com/junaozun/go -lrpxc.WithSerializationType("msgpack"),
		github.com/junaozun/go -lrpxc.WithTimeout(time.Millisecond * 2000),
	}
	s := github.com / junaozun/go-lrpxc.NewServer(opts...)
	if err := s.RegisterService("/helloworld.Greeter", new(way.Service)); err != nil {
		panic(err)
	}
	s.Serve()
}
