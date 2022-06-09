package serialization

/*
我们的框架有两种调用方式，一种是使用反射、一种是使用代码生成。
1.假如使用反射的话，我们发现只有 msgpack 、json 能够对原生的 go struct 进行序列化。由于 msgpack 性能
远高于 json，所以这里我们就直接选择了 msgpack。
2.假如使用代码生成的调用方式，这里有 gogoprotobuf、flatbuffers、thrift、protobuf 四种方案可以选。从性能上讲，四种序列化库其实性能都已经比较优秀了，
但是其中 gogoprotobuf 的序列化性能是最好的。由于目前使用最广泛的还是 protobuf，考虑到 gogoprotobuf
和 protobuf 不兼容，这里选择了比较广泛的通用化方案 protobuf
*/

type Serialization interface {
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
}

const (
	Protobuf = "protobuf" // protobuf
	MsgPack  = "msgpack"  // msgpack
	Json     = "json"     // json
)

var serializationMap = make(map[string]Serialization)

var DefaultSerialization = NewSerialization()

var NewSerialization = func() Serialization {
	return &pbSerialization{}
}

func init() {
	registerSerialization(Protobuf, DefaultSerialization)
	registerSerialization(MsgPack, &MsgpackSerialization{})
}

func registerSerialization(name string, serialization Serialization) {
	if serializationMap == nil {
		serializationMap = make(map[string]Serialization)
	}
	serializationMap[name] = serialization
}

// GetSerialization get a Serialization by a serialization name
func GetSerialization(name string) Serialization {
	if v, ok := serializationMap[name]; ok {
		return v
	}
	return DefaultSerialization
}
