package network

import (
	"reflect"
	"fmt"
	"bufio"
	"io"
	"encoding/binary"
	deserializer "github.com/anhnguyentrung/binaryserializer"
	nwtypes "bft/network/types"
	"bft/types"
	"bft/crypto"
)

const SHA256TypeSize = 32
const PublicKeySize  = 33
const SignatureSize  = 65

func UnmarshalBinary(buf []byte, v interface{}) error {
	d := deserializer.NewDeserializer(buf)
	extension := func(v interface{}) error {
		rv := reflect.Indirect(reflect.ValueOf(v))
		switch v.(type) {
		case *nwtypes.MessageType:
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
		default:
			rv := reflect.Indirect(reflect.ValueOf(v))
			return fmt.Errorf("wrong type: %s", rv.Type().String())
		}
	}
	d.Extension = extension
	return d.Deserialize(v)
}

func UnmarshalBinaryMessage(reader *bufio.Reader, message *nwtypes.Message) error {
	typeBuf := make([]byte, 1, 1)
	_, err := io.ReadFull(reader, typeBuf)
	if err != nil {
		return err
	}
	messageType := nwtypes.MessageType(typeBuf[0])
	lenBuf := make([]byte, 4, 4)
	_, err = io.ReadFull(reader, lenBuf)
	if err != nil {
		return err
	}
	length := binary.BigEndian.Uint32(lenBuf)
	messageData := make([]byte, length, length)
	n, err := io.ReadFull(reader, messageData)
	if uint32(n) != length {
		return fmt.Errorf("wrong length")
	}
	message.Header.Type = messageType
	message.Header.Length = length
	message.Payload = messageData
	return nil
}
