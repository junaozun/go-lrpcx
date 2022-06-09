package loadbalance

import (
	"sync"
	"time"

	"github.com/junaozun/go-lrpxc/selector"
)

// 轮询算法
// 这里，每个服务名和其对应的服务器列表是一个映射关系，所以我们用一个 map 结构来保存所有服务的服务名
// 和其对应的服务器列表。考虑到并发安全性，这里使用 sync.Map
type roundRobinBalancer struct {
	pickers  *sync.Map
	duration time.Duration // time duration to update again
}

func newRoundRobinBalancer() *roundRobinBalancer {
	return &roundRobinBalancer{
		pickers:  new(sync.Map),
		duration: 3 * time.Minute,
	}
}

func (r *roundRobinBalancer) Balance(serviceName string, nodes []*selector.Node) *selector.Node {

	var picker *roundRobinPicker

	if p, ok := r.pickers.Load(serviceName); !ok {
		picker = &roundRobinPicker{
			lastUpdateTime: time.Now(),
			duration:       r.duration,
			length:         len(nodes),
		}
	} else {
		picker = p.(*roundRobinPicker)
	}

	node := picker.pick(nodes)
	r.pickers.Store(serviceName, picker)
	return node
}

// 由于针对每一个服务 Service，我们还需要记录它的一些状态，比如服务列表的上次访问下标，服务列表的长度，上次访问时间等。
type roundRobinPicker struct {
	length         int           // service nodes length
	lastUpdateTime time.Time     // last update time
	duration       time.Duration // time duration to update again
	lastIndex      int           // last accessed index
}

// roundRobinPicker 真正实现了根据上次访问下标 lastIndex、服务列表长度 length、上次访问时间 lastUpdateTime 等从一个服务列表里面去获取一个服务节点
func (rp *roundRobinPicker) pick(nodes []*selector.Node) *selector.Node {
	if len(nodes) == 0 {
		return nil
	}

	// update picker after timeout
	if time.Now().Sub(rp.lastUpdateTime) > rp.duration ||
		len(nodes) != rp.length {
		rp.length = len(nodes)
		rp.lastUpdateTime = time.Now()
		rp.lastIndex = 0
	}

	if rp.lastIndex == len(nodes)-1 {
		rp.lastIndex = 0
		return nodes[0]
	}

	rp.lastIndex += 1
	return nodes[rp.lastIndex]
}
