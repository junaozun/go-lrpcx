package consul

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/junaozun/go-lrpxc/plugin"
	"github.com/junaozun/go-lrpxc/selector"
	"github.com/junaozun/go-lrpxc/selector/loadbalance"

	"github.com/hashicorp/consul/api"
)

/*
 1. 基于consul实现服务注册与发现
Service 应该去向 consul server 发起注册，consul 会将 Service 的服务名和服务地址
保存在一个 k-v 存储中。与 consul 的通信有 DNS、HTTP、gRPC 三种协议，我们采取通用的 HTTP 协议实现。
*/

/*
2、服务发现怎么实现？
因为我们采用的是客户端服务发现的模式，所以由 client 去请求 consul server，
根据某个 Service 的服务名，去 consul 的 k-v 存储中获取到服务的地址。通信的方式也是 HTTP 协议。
*/

/*
3.这里由于引入了 consul，要想使用 consul 提供的服务发现能力，我们在启动 Server 之前，要先进行
consul 的初始化，启动 consul 服务，这样才能保证 consul 能够接收我们的服务注册、服务发现的请求。
关于 consul 的初始化过程，我们将其统一放在插件初始化过程中去实现。这符合框架的插件化、可插拔的思想。
*/

type Consul struct {
	opts         *plugin.Options
	client       *api.Client
	config       *api.Config
	balancerName string // load balancing mode, including random, polling, weighted polling, consistent hash, etc
	writeOptions *api.WriteOptions
	queryOptions *api.QueryOptions
}

const Name = "consul"

// 由于 consul 是以插件化的形式实现的，所以我们同样也需要实现插件的通用接口 Plugin。
func init() {
	plugin.Register(Name, ConsulSvr)
	selector.RegisterSelector(Name, ConsulSvr)
}

var ConsulSvr = &Consul{
	opts: &plugin.Options{},
}

func (c *Consul) InitConfig() error {

	config := api.DefaultConfig()
	c.config = config

	config.HttpClient = http.DefaultClient
	config.Address = c.opts.SelectorSvrAddr
	config.Scheme = "http"

	client, err := api.NewClient(config)
	if err != nil {
		return err
	}

	c.client = client

	return nil
}

// 由于 consul 是服务发现组件，所以它需要实现服务发现的统一接口 Selector，也就是 Select 函数
// select 就分为两步了，第一步 Resolve 方法其实就是服务发现的过程，就是将所有的服务找出，然后Balance 是负载均衡实现，通过loadbanance算法找出一个节点
func (c *Consul) Select(serviceName string) (string, error) {

	nodes, err := c.Resolve(serviceName)

	if nodes == nil || len(nodes) == 0 || err != nil {
		return "", err
	}

	balancer := loadbalance.GetBalancer(c.balancerName)
	node := balancer.Balance(serviceName, nodes)

	if node == nil {
		return "", fmt.Errorf("no services find in %s", serviceName)
	}

	return parseAddrFromNode(node)
}

// Resolve 这个方法是通过一个服务名去获取服务列表
func (c *Consul) Resolve(serviceName string) ([]*selector.Node, error) {

	pairs, _, err := c.client.KV().List(serviceName, nil)
	if err != nil {
		return nil, err
	}

	if len(pairs) == 0 {
		return nil, fmt.Errorf("no services find in path : %s", serviceName)
	}
	var nodes []*selector.Node
	for _, pair := range pairs {
		nodes = append(nodes, &selector.Node{
			Key:   pair.Key,
			Value: pair.Value,
		})
	}
	return nodes, nil
}

func parseAddrFromNode(node *selector.Node) (string, error) {
	if node.Key == "" {
		return "", errors.New("addr is empty")
	}

	strs := strings.Split(node.Key, "/")

	return strs[len(strs)-1], nil
}

// consul 需要实现plugin的接口
// Server 启动的时候需要将 Service 去 consul 上进行注册，这个方法实现了服务注册的过程。
// 想将服务名和服务地址包装秤 KVPair 的形式，然后通过调用 consul api 的 Put 方法进行的将 KVPair 同步到 consul 的 k-v 存储。
func (c *Consul) Init(opts ...plugin.Option) error {

	for _, o := range opts {
		o(c.opts)
	}

	if len(c.opts.Services) == 0 || c.opts.SvrAddr == "" || c.opts.SelectorSvrAddr == "" {
		return fmt.Errorf("consul init codes, len(services) : %d, svrAddr : %s, selectorSvrAddr : %s",
			len(c.opts.Services), c.opts.SvrAddr, c.opts.SelectorSvrAddr)
	}

	if err := c.InitConfig(); err != nil {
		return err
	}

	for _, serviceName := range c.opts.Services {
		nodeName := fmt.Sprintf("%s/%s", serviceName, c.opts.SvrAddr)

		kvPair := &api.KVPair{
			Key:   nodeName,
			Value: []byte(c.opts.SvrAddr),
			Flags: api.LockFlagValue,
		}

		if _, err := c.client.KV().Put(kvPair, c.writeOptions); err != nil {
			return err
		}
	}

	return nil
}

// 我们知道，服务端 server 需要和 consul 通信来进行服务注册。那么客户端 client
// 也要和 consul 通信来进行服务发现，那么 client 如何知道 consul 地址呢？这里也需要在 client
// 进行一步 consul 初始化动作
// 客户端进行 Call 调用下游之前，需要调用 consul.Init 方法设置 consul server 的监听地址
func Init(consulSvrAddr string, opts ...plugin.Option) error {
	for _, o := range opts {
		o(ConsulSvr.opts)
	}

	ConsulSvr.opts.SelectorSvrAddr = consulSvrAddr
	err := ConsulSvr.InitConfig()
	return err
}
