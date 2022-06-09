package github

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/junaozun/go-lrpxc/codes"
	"github.com/junaozun/go-lrpxc/interceptor"
	"github.com/junaozun/go-lrpxc/metadata"
	"github.com/junaozun/go-lrpxc/protocol"
	"github.com/junaozun/go-lrpxc/serialization"
	"github.com/junaozun/go-lrpxc/transport/server_transport"
	"github.com/junaozun/go-lrpxc/utils"
)

// Service 的接口定义了每个服务需要提供的通用能力，包括 Register （处理函数 Handler 的注册）、提供服务 Serve，服务关闭 Close 等方法
type Service interface {
	Register(string, Handler)
	Serve(*ServerOptions)
	Close()
	Name() string
}

// 是 Service 接口的具体实现。它的核心是 handlers 这个 map，每一类请求会分配一个 Handler 进行处理
type service struct {
	svr         interface{}        // server
	ctx         context.Context    // 每一个 service 一个上下文进行管理
	cancel      context.CancelFunc // context 的控制器
	serviceName string             // 服务名
	handlers    map[string]Handler // 方法名：Handler
	opts        *ServerOptions     // 参数选项

	closing bool // whether the service is closing
}

// ServiceDesc is a detailed description of a service
type ServiceDesc struct {
	Svr         interface{}
	ServiceName string
	Methods     []*MethodDesc
	HandlerType interface{}
}

// MethodDesc is a detailed description of a method
type MethodDesc struct {
	MethodName string
	Handler    Handler
}

// Handler is the handler of a method
type Handler func(context.Context, interface{}, func(interface{}) error, []interceptor.ServerInterceptor) (interface{}, error)

func (s *service) Register(handlerName string, handler Handler) {
	if s.handlers == nil {
		s.handlers = make(map[string]Handler)
	}
	s.handlers[handlerName] = handler
}

// 构建 transport ，监听客户端请求，根据请求的服务名 serviceName 和请求的方法名 methodName
// 调用相应的 handler 去处理请求，然后进行回包。
func (s *service) Serve(opts *ServerOptions) {
	s.opts = opts

	transportOpts := []server_transport.ServerTransportOption{
		server_transport.WithServerAddress(s.opts.address),
		server_transport.WithServerNetwork(s.opts.network),
		server_transport.WithHandler(s),
		server_transport.WithServerTimeout(s.opts.timeout),
		server_transport.WithSerializationType(s.opts.serializationType),
		server_transport.WithProtocol(s.opts.protocol),
	}

	serverTransport := server_transport.GetServerTransport(s.opts.protocol)

	s.ctx, s.cancel = context.WithCancel(context.Background())

	if err := serverTransport.ListenAndServe(s.ctx, transportOpts...); err != nil {
		fmt.Printf("%s serve error, %v", s.opts.network, err)
		return
	}

	fmt.Printf("%s service serving at %s ... \n", s.opts.protocol, s.opts.address)

	<-s.ctx.Done()
}

func (s *service) Close() {
	s.closing = true
	if s.cancel != nil {
		s.cancel()
	}
	fmt.Println("service closing ...")
}

func (s *service) Name() string {
	return s.serviceName
}

func (s *service) Handle(ctx context.Context, reqbuf []byte) ([]byte, error) {

	// parse protocol header
	request := &protocol.Request{}
	if err := proto.Unmarshal(reqbuf, request); err != nil {
		return nil, err
	}

	ctx = metadata.WithServerMetadata(ctx, request.Metadata)

	serverSerialization := serialization.GetSerialization(s.opts.serializationType)

	dec := func(req interface{}) error {

		if err := serverSerialization.Unmarshal(request.Payload, req); err != nil {
			return err
		}
		return nil
	}

	if s.opts.timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.opts.timeout)
		defer cancel()
	}

	_, method, err := utils.ParseServicePath(string(request.ServicePath))
	if err != nil {
		return nil, codes.New(codes.ClientMsgErrorCode, "method is invalid")
	}

	handler := s.handlers[method]
	if handler == nil {
		return nil, errors.New("handlers is nil")
	}

	rsp, err := handler(ctx, s.svr, dec, s.opts.interceptors)
	if err != nil {
		return nil, err
	}

	rspbuf, err := serverSerialization.Marshal(rsp)
	if err != nil {
		return nil, err
	}

	return rspbuf, nil
}
