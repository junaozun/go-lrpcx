package plugin

import "github.com/opentracing/opentracing-go"

//这里由于服务发现插件和 tracing 的插件都有其个性化配置，导致其初始化方法的结构不一样，所以细化了
//一下 Plugin 接口，分为 ResolverPlugin 和 TracingPlugin 两类
type Plugin interface{}

type ResolverPlugin interface {
	Init(...Option) error
}

type TracingPlugin interface {
	Init(...Option) (opentracing.Tracer, error)
}

var PluginMap = make(map[string]Plugin)

// 开放注册入口给插件调用进行注册
func Register(name string, plugin Plugin) {
	if PluginMap == nil {
		PluginMap = make(map[string]Plugin)
	}
	PluginMap[name] = plugin
}

type Options struct {
	SvrAddr         string   // server address
	Services        []string // service arrays
	SelectorSvrAddr string   // server discovery address ，e.g. consul server address
	TracingSvrAddr  string   // tracing server address，e.g. jaeger server address
}

// Option provides operations on Options
type Option func(*Options)

// WithSvrAddr allows you to set SvrAddr of Options
func WithSvrAddr(addr string) Option {
	return func(o *Options) {
		o.SvrAddr = addr
	}
}

// WithSvrAddr allows you to set Services of Options
func WithServices(services []string) Option {
	return func(o *Options) {
		o.Services = services
	}
}

// WithSvrAddr allows you to set SelectorSvrAddr of Options
func WithSelectorSvrAddr(addr string) Option {
	return func(o *Options) {
		o.SelectorSvrAddr = addr
	}
}

// WithSvrAddr allows you to set TracingSvrAddr of Options
func WithTracingSvrAddr(addr string) Option {
	return func(o *Options) {
		o.TracingSvrAddr = addr
	}
}
