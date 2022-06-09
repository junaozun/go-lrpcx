package client

import (
	"time"

	"github.com/junaozun/go-lrpxc/interceptor"
	"github.com/junaozun/go-lrpxc/transport/client_transport"
)

// ClientOptions defines the client call parameters
type ClientOptions struct {
	serviceName       string        // service name
	method            string        // method name
	target            string        // format e.g.:  ip:port 127.0.0.1:8000
	timeout           time.Duration // timeout
	network           string        // network type, e.g.:  tcp、udp
	protocol          string        // protocol type , e.g. : proto、json
	serializationType string        // seralization type , e.g. : proto、msgpack
	transportOpts     client_transport.ClientTransportOptions
	interceptors      []interceptor.ClientInterceptor
	selectorName      string // service discovery name, e.g. : consul、zookeeper、etcd
	// perRPCAuth        []auth.PerRPCAuth // authentication information required for each RPC call
	// transportAuth     auth.TransportAuth
}

type ClientOption func(*ClientOptions)

func WithServiceName(serviceName string) ClientOption {
	return func(o *ClientOptions) {
		o.serviceName = serviceName
	}
}

func Del(options *ClientOptions, serverName string) {
	options.serviceName = serverName
}

func WithMethod(method string) ClientOption {
	return func(o *ClientOptions) {
		o.method = method
	}
}

func WithTarget(target string) ClientOption {
	return func(o *ClientOptions) {
		o.target = target
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(o *ClientOptions) {
		o.timeout = timeout
	}
}

func WithNetwork(network string) ClientOption {
	return func(o *ClientOptions) {
		o.network = network
	}
}

func WithProtocol(protocol string) ClientOption {
	return func(o *ClientOptions) {
		o.protocol = protocol
	}
}

func WithSerializationType(serializationType string) ClientOption {
	return func(o *ClientOptions) {
		o.serializationType = serializationType
	}
}

func WithSelectorName(selectorName string) ClientOption {
	return func(o *ClientOptions) {
		o.selectorName = selectorName
	}
}

func WithInterceptor(interceptors ...interceptor.ClientInterceptor) ClientOption {
	return func(o *ClientOptions) {
		o.interceptors = append(o.interceptors, interceptors...)
	}
}

// func WithPerRPCAuth(rpcAuth auth.PerRPCAuth) ClientOption {
//	return func(o *ClientOptions) {
//		o.perRPCAuth = append(o.perRPCAuth, rpcAuth)
//	}
// }
//
// func WithTransportAuth(transportAuth auth.TransportAuth) ClientOption {
//	return func(o *ClientOptions) {
//		o.transportAuth = transportAuth
//	}
//}
