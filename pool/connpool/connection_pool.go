package connpool

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

/*
接池是一个创建和管理连接的缓冲池技术。为什么需要连接池呢？我们知道，client 每次向 server 发起请求都会创建一
个连接。一般一个 rpc 请求消耗的时间可能是几百毫秒到几秒，也就是说在一个比较短的时间内，这个连接就会被销毁。
假设我们一秒钟需要处理 20w 个请求，假如不使用连接池的话，可能几万十几万的连接在短时间内都会被创建和销毁，
这对 cpu 资源是一个很大的消耗，同时因为我们的端口数是 1~65535，除了一些端口被计算机内部占用，每次 client
创建连接都需要分配一个端口，假如并发量过大的话，可能会出现计算机端口不够用的情况
*/
// Pool provides a pooling capability for connections, enabling connection reuse
type Pool interface {
	Get(ctx context.Context, network string, address string) (net.Conn, error)
}

var poolMap = make(map[string]Pool)
var oneByte = make([]byte, 1)

func init() {
	registorPool("default", DefaultPool)
}

func registorPool(poolName string, pool Pool) {
	poolMap[poolName] = pool
}

// GetPool get a Pool by a poolManager name
func GetPool(poolName string) Pool {
	if v, ok := poolMap[poolName]; ok {
		return v
	}
	return DefaultPool
}

// poolManager manager
type poolManager struct {
	opts  *Options
	conns *sync.Map
}

// TODO expose the ConnPool options
var DefaultPool = NewConnPool()

func NewConnPool(opt ...Option) *poolManager {
	// default options
	opts := &Options{
		maxCap:      1000,
		idleTimeout: 1 * time.Minute,
		dialTimeout: 200 * time.Millisecond,
	}
	m := &sync.Map{}

	p := &poolManager{
		conns: m,
		opts:  opts,
	}
	for _, o := range opt {
		o(p.opts)
	}

	return p
}

func (p *poolManager) Get(ctx context.Context, network string, address string) (net.Conn, error) {

	if value, ok := p.conns.Load(address); ok {
		if cp, ok := value.(*sonConnPool); ok {
			conn, err := cp.Get(ctx)
			return conn, err
		}
	}

	cp, err := p.NewSonConnPool(ctx, network, address)
	if err != nil {
		return nil, err
	}

	p.conns.Store(address, cp)

	return cp.Get(ctx)
}

func (p *poolManager) NewSonConnPool(ctx context.Context, network string, address string) (*sonConnPool, error) {
	c := &sonConnPool{
		initialCap: p.opts.initialCap,
		maxCap:     p.opts.maxCap,
		Dial: func(ctx context.Context) (net.Conn, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			timeout := p.opts.dialTimeout
			if t, ok := ctx.Deadline(); ok {
				timeout = t.Sub(time.Now())
			}

			return net.DialTimeout(network, address, timeout)
		},
		conns:       make(chan *PoolConn, p.opts.maxCap),
		idleTimeout: p.opts.idleTimeout,
		dialTimeout: p.opts.dialTimeout,
	}

	if p.opts.initialCap == 0 {
		// default initialCap is 1
		p.opts.initialCap = 1
	}

	for i := 0; i < p.opts.initialCap; i++ {
		conn, err := c.Dial(ctx)
		if err != nil {
			return nil, err
		}
		c.Put(c.wrapConn(conn))
	}

	c.RegisterChecker(3*time.Second, c.Checker)
	return c, nil
}

// 子链接池
type sonConnPool struct {
	net.Conn                  // 这个有用吗？
	initialCap  int           // initial capacity 连接池中链接的数量
	maxCap      int           // max capacity
	maxIdle     int           // max idle conn number
	idleTimeout time.Duration // idle timeout
	dialTimeout time.Duration // dial timeout
	Dial        func(context.Context) (net.Conn, error)
	conns       chan *PoolConn
	mu          sync.RWMutex
}

func (c *sonConnPool) Get(ctx context.Context) (net.Conn, error) {
	if c.conns == nil {
		return nil, ErrConnClosed
	}
	select {
	case pc := <-c.conns:
		if pc == nil {
			return nil, ErrConnClosed
		}

		if pc.unusable {
			return nil, ErrConnClosed
		}

		return pc, nil
	default:
		conn, err := c.Dial(ctx)
		if err != nil {
			return nil, err
		}
		return c.wrapConn(conn), nil
	}
}

func (c *sonConnPool) Close() {
	c.mu.Lock()
	conns := c.conns
	c.conns = nil
	c.Dial = nil
	c.mu.Unlock()

	if conns == nil {
		return
	}
	close(conns)
	for conn := range conns {
		conn.MarkUnusable()
		conn.Close()
	}
}

func (c *sonConnPool) Put(conn *PoolConn) error {
	if conn == nil {
		return errors.New("connection closed")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conns == nil {
		conn.MarkUnusable()
		conn.Close()
	}

	select {
	case c.conns <- conn:
		return nil
	default:
		// 连接池满
		return conn.Close()
	}
}

func (c *sonConnPool) RegisterChecker(internal time.Duration, checker func(conn *PoolConn) bool) {

	if internal <= 0 || checker == nil {
		return
	}

	go func() {

		for {

			time.Sleep(internal)

			length := len(c.conns)

			for i := 0; i < length; i++ {

				select {
				case pc := <-c.conns:

					if !checker(pc) {
						pc.MarkUnusable()
						pc.Close()
						break
					} else {
						c.Put(pc)
					}
				default:
					break
				}

			}
		}

	}()
}

func (c *sonConnPool) Checker(pc *PoolConn) bool {

	// check timeout
	if pc.t.Add(c.idleTimeout).Before(time.Now()) {
		return false
	}

	// check conn is alive or not
	if !isConnAlive(pc.Conn) {
		return false
	}

	return true
}

func isConnAlive(conn net.Conn) bool {
	conn.SetReadDeadline(time.Now().Add(time.Millisecond))

	if n, err := conn.Read(oneByte); n > 0 || err == io.EOF {
		return false
	}

	conn.SetReadDeadline(time.Time{})
	return true
}
