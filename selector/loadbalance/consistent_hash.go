package loadbalance

import (
	"github.com/junaozun/go-lrpxc/selector"
)

type consistHashBalancer struct {
}

// 一致性哈希算法
func newConsistentHashBalancer() *consistHashBalancer {
	return &consistHashBalancer{}
}

// todo 未实现
func (r *consistHashBalancer) Balance(serviceName string, nodes []*selector.Node) *selector.Node {
	return nil
}
