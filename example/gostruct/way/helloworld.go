package way

import "context"

type Service struct {
}

type HelloRequest struct {
	Msg string
}

type HelloReply struct {
	Msg string
}

/*
这里会通过反射的方式，通过 service 引用去获取 service 的每一个方法生成一个 handler，
然后添加到 service 的 handler map 里面。当接收到 client 发过来的请求是，通过服务的方法名获取到
这个服务的 handler，进而处理请求，得到响应，发送给客户端
*/

func (s *Service) SayHello(ctx context.Context, req *HelloRequest) (*HelloReply, error) {
	rsp := &HelloReply{
		Msg: "world",
	}

	return rsp, nil
}
