package jaeger

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/junaozun/go-lrpxc/interceptor"
	"github.com/junaozun/go-lrpxc/plugin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go/config"
)

type Jaeger struct {
	opts *plugin.Options
}

const Name = "jaeger"
const JaegerClientName = "gorpc-client-jaeger"
const JaegerServerName = "gorpc-server-jaeger"

func init() {
	plugin.Register(Name, JaegerSvr)
}

// global jaeger objects for framework
var JaegerSvr = &Jaeger{
	opts: &plugin.Options{},
}

type jaegerCarrier map[string][]byte

func (m jaegerCarrier) Set(key, val string) {
	key = strings.ToLower(key)
	m[key] = []byte(val)
}

func (m jaegerCarrier) ForeachKey(handler func(key, val string) error) error {
	for k, v := range m {
		handler(k, string(v))
	}
	return nil
}

func (j *Jaeger) Init(opts ...plugin.Option) (opentracing.Tracer, error) {

	for _, o := range opts {
		o(j.opts)
	}

	if j.opts.TracingSvrAddr == "" {
		return nil, errors.New("jaeger init codes, traingSvrAddr is empty")
	}

	return initJaeger(j.opts.TracingSvrAddr, JaegerServerName, opts...)

}

// 第一是初始化 jaeger 的配置，并且通过这些初始化配置来创建一个 tracer 实例，第二是将这个 tracer 实例作为
// opentracing 规范的实现。之前我们说到了 opentracing 只是一套规范，并没有进行链路追踪的具体实现，
// 所以无论你是使用 jaeger 还是 zipkin 等其他链路追踪系统，你都需要进行 opentracing.SetGlobalTracer(tracer) ，
// 将 tracer 设为 opentracing 的真正实现。
func initJaeger(tracingSvrAddr string, jaegerServiceName string, opts ...plugin.Option) (opentracing.Tracer, error) {
	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "const", // Fixed sampling
			Param: 1,       // 1= full sampling, 0= no sampling
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: tracingSvrAddr,
		},
		ServiceName: jaegerServiceName,
	}

	tracer, _, err := cfg.NewTracer()
	if err != nil {
		return nil, err
	}

	opentracing.SetGlobalTracer(tracer)

	return tracer, err
}

// 客户端jaeger的初始化
func Init(tracingSvrAddr string, opts ...plugin.Option) (opentracing.Tracer, error) {
	return initJaeger(tracingSvrAddr, JaegerClientName, opts...)
}

// client 端上报 span，主要的步骤是，先通过 opentracing.SpanFromContext 获取上游带下来的 span 上下文信息，接着调用 tracer.StartSpan 创建一个 client span，
// 通过调用 tracer.Inject，将所需要透传给下游的一些信息塞到 Span 里面。jaegerCarrier 是一种 map[string] []byte 结构，用来作为传输一些 key-value 数据的载体
func OpenTracingClientInterceptor(tracer opentracing.Tracer, spanName string) interceptor.ClientInterceptor {

	return func(ctx context.Context, req, rsp interface{}, ivk interceptor.Invoker) error {

		// var parentCtx opentracing.SpanContext
		//
		// if parent := opentracing.SpanFromContext(ctx); parent != nil {
		//	parentCtx = parent.Context()
		// }

		// clientSpan := tracer.StartSpan(spanName, ext.SpanKindRPCClient, opentracing.ChildOf(parentCtx))
		clientSpan := tracer.StartSpan(spanName, ext.SpanKindRPCClient)
		defer clientSpan.Finish()

		mdCarrier := &jaegerCarrier{}

		if err := tracer.Inject(clientSpan.Context(), opentracing.HTTPHeaders, mdCarrier); err != nil {
			clientSpan.LogFields(log.String("event", "Tracer.Inject() failed"), log.Error(err))
		}

		clientSpan.LogFields(log.String("spanName", spanName))

		return ivk(ctx, req, rsp)

	}
}

// server 端上报 span，主要的步骤是，先调用 tracer.Extract 解析 Span 的上下文信息，获得一个 SpanContext，接着调用 tracer.StartSpan
// 进行创建一个 server span，并且把 server span 放到上下文 context 中进行透传，jeager 会自动对 span 进行上报
func OpenTracingServerInterceptor(tracer opentracing.Tracer, spanName string) interceptor.ServerInterceptor {

	return func(ctx context.Context, req interface{}, handler interceptor.Handler) (interface{}, error) {

		mdCarrier := &jaegerCarrier{}

		spanContext, err := tracer.Extract(opentracing.HTTPHeaders, mdCarrier)
		if err != nil && err != opentracing.ErrSpanContextNotFound {
			return nil, errors.New(fmt.Sprintf("tracer extract codes : %v", err))
		}
		serverSpan := tracer.StartSpan(spanName, ext.RPCServerOption(spanContext), ext.SpanKindRPCServer)
		defer serverSpan.Finish()

		ctx = opentracing.ContextWithSpan(ctx, serverSpan)

		serverSpan.LogFields(log.String("spanName", spanName))

		return handler(ctx, req)
	}

}
