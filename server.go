package github

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"syscall"

	"github.com/junaozun/go-lrpxc/interceptor"
	"github.com/junaozun/go-lrpxc/plugin"
	"github.com/junaozun/go-lrpxc/plugin/jaeger"
)

/*
server
要搭建 server 层，首先我们要明确 server 层需要支持哪些能力，其实 server 的核心就是提供服务请求的处理能力。
server 侧定义服务，发布服务，接收到服务的请求后，根据服务名和请求的方法名去路由到一个 handler 处理器，然后由 handler 处理请求，
得到响应，并且把响应数据发送给 client。
*/

type Server struct {
	opts    *ServerOptions  // 选项模型，用来透传业务自己指定的一些参数，比如服务监听的地址 address，网络类型 network 是 tcp 还是 udp，后端服务的超时时间 timeout 等。
	service Service         // 每个 Service 表示一个服务，一个 server 可以发布多个服务，用服务名 serviceName 作 map 的 key
	plugins []plugin.Plugin // Server 中添加 plugins 成员变量，它是一个插件数组。
	closing bool            // whether the server is closing
}

func NewServer(opt ...ServerOption) *Server {
	s := &Server{
		opts: &ServerOptions{},
	}

	s.service = NewService(s.opts)

	for _, o := range opt {
		o(s.opts)
	}
	// 当调用 server.New 函数时，遍历插件 PluginMap，将所有插件 Plugin 添加到 plugins 中去
	for pluginName, plugin := range plugin.PluginMap {
		if !containPlugin(pluginName, s.opts.pluginNames) {
			continue
		}
		s.plugins = append(s.plugins, plugin)
	}
	return s
}

func NewService(opts *ServerOptions) Service {
	return &service{
		opts: opts,
	}
}

func containPlugin(pluginName string, plugins []string) bool {
	for _, plugin := range plugins {
		if pluginName == plugin {
			return true
		}
	}
	return false
}

func (s *Server) Serve() {
	// 在调用 Server.Serve() 方法时，在 server 中的所有 service 提供服务之前，调用 InitPlugins 方法进行插件的配置初始化。
	err := s.InitPlugins()
	if err != nil {
		panic(err)
	}
	// 遍历 service map 里面所有的 service，然后运行 service 的 Serve 方法
	// for _, service := range s.service {
	//	go service.Serve(s.opts)
	// }

	s.service.Serve(s.opts)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGSEGV)
	<-ch

	s.Close()
}

func (s *Server) Close() {
	s.closing = false
	s.service.Close()
}

func (s *Server) InitPlugins() error {
	// init plugins
	for _, p := range s.plugins {

		switch val := p.(type) {

		case plugin.ResolverPlugin:
			var services []string
			services = append(services, s.service.Name())

			pluginOpts := []plugin.Option{
				plugin.WithSelectorSvrAddr(s.opts.selectorSvrAddr),
				plugin.WithSvrAddr(s.opts.address),
				plugin.WithServices(services),
			}
			if err := val.Init(pluginOpts...); err != nil {
				fmt.Printf("resolver init codes, %v", err)
				return err
			}

		case plugin.TracingPlugin:

			pluginOpts := []plugin.Option{
				plugin.WithTracingSvrAddr(s.opts.tracingSvrAddr),
			}

			tracer, err := val.Init(pluginOpts...)
			if err != nil {
				fmt.Printf("tracing init codes, %v", err)
				return err
			}

			s.opts.interceptors = append(s.opts.interceptors, jaeger.OpenTracingServerInterceptor(tracer, s.opts.tracingSpanName))

		default:

		}

	}

	return nil
}

type emptyInterface interface{}

func (s *Server) RegisterService(serviceName string, svr interface{}) error {

	svrType := reflect.TypeOf(svr)
	svrValue := reflect.ValueOf(svr)

	sd := &ServiceDesc{
		ServiceName: serviceName,
		// for compatibility with code generation
		HandlerType: (*emptyInterface)(nil),
		Svr:         svr,
	}

	methods, err := getServiceMethods(svrType, svrValue)
	if err != nil {
		return err
	}

	sd.Methods = methods

	s.Register(sd, svr)

	return nil
}

func getServiceMethods(serviceType reflect.Type, serviceValue reflect.Value) ([]*MethodDesc, error) {

	var methods []*MethodDesc

	for i := 0; i < serviceType.NumMethod(); i++ {
		method := serviceType.Method(i)

		if err := checkMethod(method.Type); err != nil {
			return nil, err
		}

		methodHandler := func(ctx context.Context, svr interface{}, dec func(interface{}) error, ceps []interceptor.ServerInterceptor) (interface{}, error) {

			reqType := method.Type.In(2)

			// determine type
			req := reflect.New(reqType.Elem()).Interface()

			if err := dec(req); err != nil {
				return nil, err
			}

			if len(ceps) == 0 {
				values := method.Func.Call([]reflect.Value{serviceValue, reflect.ValueOf(ctx), reflect.ValueOf(req)})
				// determine error
				return values[0].Interface(), nil
			}

			handler := func(ctx context.Context, reqbody interface{}) (interface{}, error) {

				values := method.Func.Call([]reflect.Value{serviceValue, reflect.ValueOf(ctx), reflect.ValueOf(req)})

				return values[0].Interface(), nil
			}

			return interceptor.ServerIntercept(ctx, req, ceps, handler)
		}

		methods = append(methods, &MethodDesc{
			MethodName: method.Name,
			Handler:    methodHandler,
		})
	}

	return methods, nil
}

func (s *Server) Register(sd *ServiceDesc, svr interface{}) {
	if sd == nil || svr == nil {
		return
	}
	ht := reflect.TypeOf(sd.HandlerType).Elem()
	st := reflect.TypeOf(svr)
	if !st.Implements(ht) {
		log.Fatalf("handlerType %v not match service : %v ", ht, st)
	}

	ser := &service{
		svr:         svr,
		serviceName: sd.ServiceName,
		handlers:    make(map[string]Handler),
	}

	for _, method := range sd.Methods {
		ser.handlers[method.MethodName] = method.Handler
	}

	s.service = ser
}

func checkMethod(method reflect.Type) error {

	// params num must >= 2 , needs to be combined with itself
	if method.NumIn() < 3 {
		return fmt.Errorf("method %s invalid, the number of params < 2", method.Name())
	}

	// return values nums must be 2
	if method.NumOut() != 2 {
		return fmt.Errorf("method %s invalid, the number of return values != 2", method.Name())
	}

	// the first parameter must be context
	ctxType := method.In(1)
	var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	if !ctxType.Implements(contextType) {
		return fmt.Errorf("method %s invalid, first param is not context", method.Name())
	}

	// the second parameter type must be pointer
	argType := method.In(2)
	if argType.Kind() != reflect.Ptr {
		return fmt.Errorf("method %s invalid, req type is not a pointer", method.Name())
	}

	// the first return type must be a pointer
	replyType := method.Out(0)
	if replyType.Kind() != reflect.Ptr {
		return fmt.Errorf("method %s invalid, reply type is not a pointer", method.Name())
	}

	// The second return value must be an error
	errType := method.Out(1)
	var errorType = reflect.TypeOf((*error)(nil)).Elem()
	if !errType.Implements(errorType) {
		return fmt.Errorf("method %s invalid, returns %s , not error", method.Name(), errType.Name())
	}

	return nil
}
