package encoding

import (
	serializer "github.com/anhnguyentrung/binaryserializer"
	"fmt"
	"reflect"
	"bft/types"
	"bft/crypto"
	"time"
	"encoding/binary"
)

func MarshalBinary(v interface{}) ([]byte, error) {
	s := serializer.NewSerializer()
	extension := func(v interface{}) error {
		switch t := v.(type) {
		case types.MessageType:
			return s.WriteBytes([]byte{byte(t)})
		case types.VoteType:
			return s.WriteBytes([]byte{byte(t)})
		case types.Hash:
			return s.WriteBytes(t[:])
		case crypto.Signature:
			if len(t.Data) != 65 {
				return fmt.Errorf("length of signature data is not 65 bytes")
			}
			return s.WriteBytes(t.Data)
		case crypto.PublicKey:
			if len(t.Data) != 33 {
				return fmt.Errorf("length of public key is not 33 bytes")
			}
			return s.WriteBytes(t.Data)
		case time.Time:
			n := uint64(t.UnixNano())
			bytes := make([]byte, serializer.Uint64Size)
			binary.BigEndian.PutUint64(bytes, n)
			return s.WriteBytes(bytes)
		default:
			rv := reflect.Indirect(reflect.ValueOf(v))
			return fmt.Errorf("wrong type: %s", rv.Type().String())
		}
	}
	s.Extension = extension
	err := s.Serialize(v)
	return s.Bytes(), err
}