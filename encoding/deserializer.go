package encoding

import (
	"reflect"
	"fmt"
	"encoding/binary"
	deserializer "github.com/anhnguyentrung/binaryserializer"
	"bft/types"
	"bft/crypto"
	"time"
)

const SHA256TypeSize = 32
const PublicKeySize  = 33
const SignatureSize  = 65

func UnmarshalBinary(buf []byte, v interface{}) error {
	d := deserializer.NewDeserializer(buf)
	extension := func(v interface{}) error {
		rv := reflect.Indirect(reflect.ValueOf(v))
		switch v.(type) {
		case *types.MessageType:
			bytes, err := d.ReadBytes(1)
			if err != nil {
				return err
			}
			rv.SetUint(uint64(bytes[0]))
			return nil
		case *types.Hash:
			bytes, err := d.ReadBytes(SHA256TypeSize)
			if err != nil {
				return err
			}
			hash := types.Hash{}
			copy(hash[:], bytes)
			rv.Set(reflect.ValueOf(hash))
			return nil
		case *crypto.PublicKey:
			bytes, err := d.ReadBytes(PublicKeySize)
			if err != nil {
				return err
			}
			publicKey := crypto.PublicKey{ Data: bytes}
			rv.Set(reflect.ValueOf(publicKey))
			return nil
		case *crypto.Signature:
			bytes, err := d.ReadBytes(SignatureSize)
			if err != nil {
				return err
			}
			signature := crypto.Signature{ Data: bytes}
			rv.Set(reflect.ValueOf(signature))
			return nil
		case *time.Time:
			bytes, err := d.ReadBytes(deserializer.Uint64Size)
			if err != nil {
				return err
			}
			unixNano := binary.BigEndian.Uint64(bytes)
			ts := time.Unix(0, int64(unixNano))
			rv.Set(reflect.ValueOf(ts))
			return nil
		default:
			rv := reflect.Indirect(reflect.ValueOf(v))
			return fmt.Errorf("wrong type: %s", rv.Type().String())
		}
	}
	d.Extension = extension
	return d.Deserialize(v)
}

func UnmarshalBinaryMessage(buf []byte, message *types.Message) error {
	pos := 0
	if len(buf) < 1 {
		return fmt.Errorf("can't read message type")
	}
	typeBuf := buf[pos:1]
	pos += 1
	messageType := types.MessageType(typeBuf[0])
	if len(buf) < pos + 4 {
		return fmt.Errorf("can't read message length")
	}
	lenBuf := buf[pos:pos+4]
	pos += 4
	length := binary.BigEndian.Uint32(lenBuf)
	if len(buf) < pos + int(length) {
		return fmt.Errorf("can't read message data")
	}
	payload := buf[pos:pos + int(length)]
	message.Header.Type = messageType
	message.Header.Length = length
	message.Payload = payload
	return nil
}
