package loadbalance

import "github.com/junaozun/go-lrpxc/selector"

type Balancer interface {
	Balance(string, []*selector.Node) *selector.Node
}

var balancerMap = make(map[string]Balancer, 0)

const (
	Random             = "random"
	RoundRobin         = "roundRobin"
	WeightedRoundRobin = "weightedRoundRobin"
	ConsistentHash     = "consistentHash"
)

func init() {
	RegisterBalancer(Random, RandomBalancer)
	RegisterBalancer(RoundRobin, RRBalancer)
	RegisterBalancer(WeightedRoundRobin, WRRBalancer)
	RegisterBalancer(ConsistentHash, ConsistHashBalancer)
}

// 请求随机分配到各个服务器。
var RandomBalancer = newRandomBalancer()

// 将所有请求依次分发到每台服务器上，适合服务器硬件配置相同的场景。
var RRBalancer = newRoundRobinBalancer()

// 根据服务器硬件配置和负载能力，为服务器设置不同的权重，在进行请求分发时，不同权重的服务器分配的流量不同，权重越大的服务器，分配的请求数越多，流量越大。
var WRRBalancer = newWeightedRoundRobinBalancer()

// 按照一致性哈希算法，将请求分发到每台服务器上。
// 优点：能够避免传统哈希算法因服务器数量变化而引起的集群雪崩问题。
// 缺点：实现较为复杂，请求量比较小的场景下，可能会出现某个服务器节点完全空闲的情况
var ConsistHashBalancer = newConsistentHashBalancer()

func RegisterBalancer(name string, balancer Balancer) {
	if balancerMap == nil {
		balancerMap = make(map[string]Balancer)
	}
	balancerMap[name] = balancer
}

// GetBalancer get a Balancer by a balancer name
func GetBalancer(name string) Balancer {
	if balancer, ok := balancerMap[name]; ok {
		return balancer
	}
	return RandomBalancer
}
