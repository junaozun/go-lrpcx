package transport

import (
	"context"
	"encoding/binary"
	"io"
	"net"

	"github.com/junaozun/go-lrpxc/codec"
	"github.com/junaozun/go-lrpxc/codes"
	"github.com/junaozun/go-lrpxc/transport/client_transport"
	"github.com/junaozun/go-lrpxc/transport/server_transport"
)

/*
   transport 作为传输层，分为 client 发送方和 server 接收方两种类型
*/

const DefaultPayloadLength = 1024
const MaxPayloadLength = 4 * 1024 * 1024

// server 传输层主要提供一种监听和处理请求的能力
type ServerTransport interface {
	ListenAndServe(context.Context, ...server_transport.ServerTransportOption) error
}

// client 传输层主要提供一种向下游发送请求的能力
type ClientTransport interface {
	Send(context.Context, []byte, ...client_transport.ClientTransportOption) ([]byte, error)
}

// 从网络流中读取数据
type Framer interface {
	ReadFrame(net.Conn) ([]byte, error)
}

type framer struct {
	buffer  []byte
	counter int // 避免死循环
}

// Create a Framer
func NewFramer() Framer {
	return &framer{
		buffer: make([]byte, DefaultPayloadLength),
	}
}

func (f *framer) Resize() {
	f.buffer = make([]byte, len(f.buffer)*2)
}

func (f *framer) ReadFrame(conn net.Conn) ([]byte, error) {
	// 帧头
	frameHeader := make([]byte, codec.FrameHeadLen)
	if num, err := io.ReadFull(conn, frameHeader); num != codec.FrameHeadLen || err != nil {
		return nil, err
	}

	// validate magic
	if magic := uint8(frameHeader[0]); magic != codec.Magic {
		return nil, codes.NewFrameworkError(codes.ClientMsgErrorCode, "invalid magic...")
	}

	// 7-11 表示包头+包体的长度
	length := binary.BigEndian.Uint32(frameHeader[7:11])

	// 规定了请求包头+包体的大小
	if length > MaxPayloadLength {
		return nil, codes.NewFrameworkError(codes.ClientMsgErrorCode, "payload too large...")
	}

	// buffer 是用一块默认的固定内存 1024 byte，来避免每次读取数据帧都需要创建和销毁内存的开销。当内存不够时，会扩容成原来的两倍
	// 为了避免包过大时或者其他不可知意外造成死循环，这里加了一个 counter 计数器。当 buffer > 4M 时或者 扩容的次数 counter 大于 12 时，会跳出循环，不再 Resize
	for uint32(len(f.buffer)) < length && f.counter <= 12 {
		f.Resize()
		f.counter++
	}

	// 从链接中读出包头+包体，存入buffer中
	if num, err := io.ReadFull(conn, f.buffer[:length]); uint32(num) != length || err != nil {
		return nil, err
	}

	// 将帧头+包头+包体 拼在一起返回
	return append(frameHeader, f.buffer[:length]...), nil
}

type ConnWrapper struct {
	net.Conn
	Framer Framer
}

func WrapConn(rawConn net.Conn) *ConnWrapper {
	return &ConnWrapper{
		Conn:   rawConn,
		Framer: NewFramer(),
	}
}
