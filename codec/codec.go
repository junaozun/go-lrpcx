package codec

import (
	"bytes"
	"encoding/binary"
	"sync"

	"github.com/golang/protobuf/proto"
)

/*
一般的协议结构会分为帧头、协议包头、协议包体三个部分，帧头一般都是起到传输控制的作用，协议包头一般是用来传输一些需要
在 client 和 server 之间进行透传的一些数据结构，比如序列化的方式、染色 key 等等。协议包体则是 client 发送的请求
数据的二进制流或者是 server 返回的响应数据的二进制流。为了减少数据包的大小，一般尽可能节约帧头长度。协议形式：
		帧头(15 byte)   +   包头    +    包体
帧头：魔数(1 byte)+版本号(1 byte)+消息类型(1 byte)+请求类型(1 byte)+是否压缩(1 byte)+流ID(2 byte) + 消息长度(4 byte) + 保留位(4 byte)
*/

type Codec interface {
	Encode([]byte) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

const FrameHeadLen = 15
const Magic = 0x11
const Version = 0

type FrameHeader struct {
	Magic        uint8  //
	Version      uint8  //
	MsgType      uint8  // 要是用来区分普通消息和心跳消息。我们用 0x0 来表示普通消息，用 0x1 来表示心跳消息，客户端向服务端发送心跳包表示自己还是存活的
	ReqType      uint8  // 用 0x0 来表示一发一收，0x1 来表示只发不收，0x2 表示客户端流式请求，0x3 表示服务端流式请求，0x4 表示双向流式请求。
	CompressType uint8  // //client 和 server 会根据这个标志位决定对传输的数据是否进行压缩/解压处理。0x0 默认不压缩，0x1 压缩
	StreamID     uint16 // stream ID //为了支持后续流式传输的能力
	Length       uint32 // total packet length ,只是包头和包体，不包括帧头
	Reserved     uint32 // 4 bytes reserved //保留位，方便后续协议进行扩展
}

func GetCodec(name string) Codec {
	if codec, ok := codecMap[name]; ok {
		return codec
	}
	return DefaultCodec
}

var codecMap = make(map[string]Codec)

var DefaultCodec = NewCodec()

// NewCodec returns a globally unique codec
var NewCodec = func() Codec {
	return &defaultCodec{}
}

func init() {
	RegisterCodec("proto", DefaultCodec)
}

// RegisterCodec registers a codec, which will be added to codecMap
func RegisterCodec(name string, codec Codec) {
	if codecMap == nil {
		codecMap = make(map[string]Codec)
	}
	codecMap[name] = codec
}

type defaultCodec struct{}

func (c *defaultCodec) Encode(data []byte) ([]byte, error) {

	totalLen := FrameHeadLen + len(data)
	buffer := bytes.NewBuffer(make([]byte, 0, totalLen))

	frame := FrameHeader{
		Magic:        Magic,
		Version:      Version,
		MsgType:      0x0,
		ReqType:      0x0,
		CompressType: 0x0,
		Length:       uint32(len(data)),
	}

	if err := binary.Write(buffer, binary.BigEndian, frame.Magic); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, frame.Version); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, frame.MsgType); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, frame.ReqType); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, frame.CompressType); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, frame.StreamID); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, frame.Length); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, frame.Reserved); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, data); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (c *defaultCodec) Decode(frame []byte) ([]byte, error) {
	return frame[FrameHeadLen:], nil
}

var bufferPool = &sync.Pool{
	New: func() interface{} {
		return &cachedBuffer{
			Buffer:            proto.Buffer{},
			lastMarshaledSize: 16,
		}
	},
}

type cachedBuffer struct {
	proto.Buffer
	lastMarshaledSize uint32
}
