package loadbalance

import (
	"sync"
	"time"

	"github.com/junaozun/go-lrpxc/selector"
)

// 加权轮训
// 基于平滑的加权轮训算法
type weightedRoundRobinBalancer struct {
	pickers  *sync.Map
	duration time.Duration // time duration to update again
}

func newWeightedRoundRobinBalancer() *weightedRoundRobinBalancer {
	return &weightedRoundRobinBalancer{
		pickers:  new(sync.Map),
		duration: 3 * time.Minute,
	}
}

func (w *weightedRoundRobinBalancer) Balance(serviceName string, nodes []*selector.Node) *selector.Node {
	var picker *wRoundRobinPicker

	if p, ok := w.pickers.Load(serviceName); !ok {
		picker = &wRoundRobinPicker{
			lastUpdateTime: time.Now(),
			duration:       w.duration,
			nodes:          getWeightedNode(nodes),
		}
		w.pickers.Store(serviceName, picker)
	} else {
		picker = p.(*wRoundRobinPicker)
	}

	node := picker.pick(nodes)
	w.pickers.Store(serviceName, picker)
	return node
}

type wRoundRobinPicker struct {
	nodes          []*weightedNode // service nodes
	lastUpdateTime time.Time       // last update time
	duration       time.Duration   // time duration to update again
}

type weightedNode struct {
	node            *selector.Node
	weight          int
	effectiveWeight int
	currentWeight   int
}

func (wr *wRoundRobinPicker) pick(nodes []*selector.Node) *selector.Node {
	if len(nodes) == 0 {
		return nil
	}

	// update picker after timeout
	if time.Now().Sub(wr.lastUpdateTime) > wr.duration ||
		len(nodes) != len(wr.nodes) {
		wr.nodes = getWeightedNode(nodes)
		wr.lastUpdateTime = time.Now()
	}

	totalWeight := 0
	maxWeight := 0
	index := 0
	for i, node := range wr.nodes {
		node.currentWeight += node.weight
		totalWeight += node.weight
		if node.currentWeight > maxWeight {
			maxWeight = node.currentWeight
			index = i
		}
	}

	wr.nodes[index].currentWeight -= totalWeight

	return wr.nodes[index].node

}

func getWeightedNode(nodes []*selector.Node) []*weightedNode {

	var wgs []*weightedNode
	for _, node := range nodes {
		wgs = append(wgs, &weightedNode{
			node:            node,
			weight:          node.Weight,
			currentWeight:   node.Weight,
			effectiveWeight: node.Weight,
		})
	}

	return wgs
}
