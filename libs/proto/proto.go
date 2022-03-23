package proto

import "github.com/golang/protobuf/proto"

func Marshal(m proto.Message) ([]byte, error) {
	buf := proto.NewBuffer(nil)
	buf.SetDeterministic(true)
	if err := buf.Marshal(m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Unmarshal(b []byte, m proto.Message) error {
	return proto.Unmarshal(b, m)
}
