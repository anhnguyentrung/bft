package serializer

import (
	"encoding/binary"
	"reflect"
	"fmt"
	"bytes"
)

const Uint16Size = 2
const Uint32Size = 4
const Uint64Size = 8

type Serializer struct {
	writer 		*bytes.Buffer
	pos			int
	Extension 	func (v interface{}) error
}

func NewSerializer() *Serializer {
	return &Serializer{
		writer:		new(bytes.Buffer),
		pos:		0,
		Extension: nil,
	}
}

func (s *Serializer) Serialize(v interface{}) error {
	switch t := v.(type) {
	case byte:
		return s.byteSerializer(t)
	case int8:
		return s.int8Serializer(t)
	case int16:
		return s.int16Serializer(t)
	case uint16:
		return s.uint16Serializer(t)
	case int32:
		return s.int32Serializer(t)
	case uint32:
		return s.uint32Serializer(t)
	case int64:
		return s.int64Serializer(t)
	case uint64:
		return s.uint64Serializer(t)
	case bool:
		return s.boolSerializer(t)
	case []byte:
		return s.bytesSerializer(t)
	case string:
		return s.stringSerializer(t)
	default:
		return s.recursiveSerializer(v)
	}
}

func (s *Serializer) byteSerializer(v byte) error {
	bytes := []byte{v}
	return s.WriteBytes(bytes)
}

func (s *Serializer) bytesSerializer(v []byte) error {
	err := s.writeLength(len(v))
	if err != nil {
		return err
	}
	return s.WriteBytes(v)
}

func (s *Serializer) int8Serializer(v int8) error {
	return s.byteSerializer(byte(v))
}

func (s *Serializer) int16Serializer(v int16) error {
	return s.uint16Serializer(uint16(v))
}

func (s *Serializer) uint16Serializer(v uint16) error {
	bytes := make([]byte, Uint16Size)
	binary.BigEndian.PutUint16(bytes, v)
	return s.WriteBytes(bytes)
}

func (s *Serializer) int32Serializer(v int32) error {
	return s.uint32Serializer(uint32(v))
}

func (s *Serializer) uint32Serializer(v uint32) error {
	bytes := make([]byte, Uint32Size)
	binary.BigEndian.PutUint32(bytes, v)
	return s.WriteBytes(bytes)
}

func (s *Serializer) int64Serializer(v int64) error {
	return s.uint64Serializer(uint64(v))
}

func (s *Serializer) uint64Serializer(v uint64) error {
	bytes := make([]byte, Uint64Size)
	binary.BigEndian.PutUint64(bytes, v)
	return s.WriteBytes(bytes)
}

func (s *Serializer) boolSerializer(v bool) error {
	bytes := []byte{0}
	if v {
		bytes[0] = 1
	}
	return s.WriteBytes(bytes)
}

func (s *Serializer) stringSerializer(v string) error {
	bytes := []byte(v)
	return s.bytesSerializer(bytes)
}

func (s *Serializer) WriteBytes(bytes []byte) error {
	s.pos += len(bytes)
	_, err := s.writer.Write(bytes)
	return err
}

func (s *Serializer) writeLength(len int) error {
	buf := make([]byte, Uint64Size)
	n := binary.PutUvarint(buf, uint64(len))
	return s.WriteBytes(buf[:n])
}

// this function is used to serialize struct, map, array and extensions
func (s *Serializer) recursiveSerializer(v interface{}) error {
	reflectValue := reflect.Indirect(reflect.ValueOf(v))
	switch reflectValue.Kind() {
	case reflect.Struct:
		return s.structSerializer(reflectValue)
	case reflect.Array:
		return s.arraySerializer(reflectValue)
	case reflect.Slice:
		return s.sliceSerializer(reflectValue)
	case reflect.Map:
		return s.mapSerializer(reflectValue)
	default:
		if s.Extension != nil {
			return s.Extension(v)
		}
		return fmt.Errorf("wrong type: %s", reflectValue.Type().String())
	}
}

func (s *Serializer) structSerializer(v reflect.Value) error {
	numField := v.NumField()
	for i:=0; i < numField; i++ {
		field := v.Field(i)
		if field.CanInterface() {
			if err := s.Serialize(field.Interface()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Serializer) arraySerializer(v reflect.Value) error {
	n := v.Len()
	for i:= 0; i < n; i++ {
		if err := s.Serialize(v.Index(i).Interface()); err != nil {
			return err
		}
	}
	return nil
}

func (s *Serializer) sliceSerializer(v reflect.Value) error {
	n := v.Len()
	if err := s.writeLength(n); err != nil {
		return err
	}
	for i:= 0; i < n; i++ {
		if err := s.Serialize(v.Index(i).Interface()); err != nil {
			return err
		}
	}
	return nil
}

func (s *Serializer) mapSerializer(v reflect.Value) error {
	n := v.Len()
	if err := s.writeLength(n); err != nil {
		return err
	}
	for _, key := range v.MapKeys() {
		value := v.MapIndex(key)
		if err := s.Serialize(key.Interface()); err != nil {
			return err
		}
		if err := s.Serialize(value.Interface()); err != nil {
			return err
		}
	}
	return nil
}

func (s *Serializer) Bytes() []byte {
	return s.writer.Bytes()
}

func MarshalBinary(v interface{}) ([]byte, error) {
	serializer := NewSerializer()
	err := serializer.Serialize(v)
	return serializer.Bytes(), err
}


