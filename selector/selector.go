package selector

//Selector 通过服务发现和负载均衡获取服务节点
type Selector interface {
	Select(string) (string, error)
}

type defaultSelector struct {
}

type Options struct {
}

type Option func(*Options)

func init() {
	RegisterSelector("default", DefaultSelector)
}

// 默认的 Selector
var DefaultSelector = &defaultSelector{}

var selectorMap = make(map[string]Selector)

// RegisterSelector supports business custom registered Selector
func RegisterSelector(name string, selector Selector) {
	if selectorMap == nil {
		selectorMap = make(map[string]Selector)
	}
	selectorMap[name] = selector
}

func (d *defaultSelector) Select(serviceName string) (string, error) {

	return "", nil
}

// GetSelector get a selector by a given selector name
func GetSelector(name string) Selector {
	if selector, ok := selectorMap[name]; ok {
		return selector
	}
	return DefaultSelector
}

type Node struct {
	Key    string
	Value  []byte
	Weight int
}
