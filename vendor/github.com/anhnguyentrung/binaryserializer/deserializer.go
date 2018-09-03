package serializer

import (
	"reflect"
	"fmt"
	"encoding/binary"
)

type Deserializer struct {
	buffer 		[]byte
	pos 		int
	Extension 	func (v interface{}) error
}

func NewDeserializer(buf []byte) *Deserializer {
	return &Deserializer{
		buffer:		buf,
		pos:		0,
		Extension:	nil,
	}
}

func (d *Deserializer) Deserialize(v interface{}) error {
	reflectValue := reflect.Indirect(reflect.ValueOf(v))
	if !reflectValue.CanAddr() {
		return fmt.Errorf("not pointer type")
	}
	if reflectValue.Type().Kind() == reflect.Ptr {
		elem := reflectValue.Type().Elem()
		reflectValue = reflect.Indirect(reflect.New(elem))
	}
	switch v.(type) {
	case *byte:
		return d.byteDeserializer(reflectValue)
	case *int16:
		return d.int16Deserializer(reflectValue)
	case *uint16:
		return d.uint16Deserializer(reflectValue)
	case *int32:
		return d.int32Deserializer(reflectValue)
	case *uint32:
		return d.uint32Deserializer(reflectValue)
	case *int64:
		return d.int64Deserializer(reflectValue)
	case *uint64:
		return d.uint64Deserializer(reflectValue)
	case *bool:
		return d.boolDeserializer(reflectValue)
	case *[]byte:
		return d.bytesDeserializer(reflectValue)
	case *string:
		return d.stringDeserializer(reflectValue)
	default:
		return d.recursiveDeserializer(v, reflectValue)
	}
}

func (d *Deserializer) ReadBytes(l int) ([]byte, error)  {
	if err := d.checkBufferLength(l); err != nil {
		return nil, err
	}
	bytes := d.buffer[d.pos : d.pos + l]
	d.pos += l
	return bytes, nil
}

func (d *Deserializer) byteDeserializer(v reflect.Value) error {
	if err := d.checkBufferLength(1); err != nil {
		return err
	}
	value := d.buffer[d.pos]
	d.pos += 1
	v.SetUint(uint64(value))
	return nil
}

func (d *Deserializer) int16Deserializer(v reflect.Value) error {
	if err := d.checkBufferLength(Uint16Size); err != nil {
		return err
	}
	value := int16(binary.BigEndian.Uint16(d.buffer[d.pos:]))
	d.pos += Uint16Size
	v.SetInt(int64(value))
	return nil
}

func (d *Deserializer) uint16Deserializer(v reflect.Value) error {
	if err := d.checkBufferLength(Uint16Size); err != nil {
		return err
	}
	value := binary.BigEndian.Uint16(d.buffer[d.pos:])
	d.pos += Uint16Size
	v.SetUint(uint64(value))
	return nil
}

func (d *Deserializer) int32Deserializer(v reflect.Value) error {
	if err := d.checkBufferLength(Uint32Size); err != nil {
		return err
	}
	value := int32(binary.BigEndian.Uint32(d.buffer[d.pos:]))
	d.pos += Uint32Size
	v.SetInt(int64(value))
	return nil
}

func (d *Deserializer) uint32Deserializer(v reflect.Value) error {
	if err := d.checkBufferLength(Uint32Size); err != nil {
		return err
	}
	value := binary.BigEndian.Uint32(d.buffer[d.pos:])
	d.pos += Uint32Size
	v.SetUint(uint64(value))
	return nil
}

func (d *Deserializer) int64Deserializer(v reflect.Value) error {
	if err := d.checkBufferLength(Uint64Size); err != nil {
		return err
	}
	value := int64(binary.BigEndian.Uint64(d.buffer[d.pos:]))
	d.pos += Uint64Size
	v.SetInt(value)
	return nil
}

func (d *Deserializer) uint64Deserializer(v reflect.Value) error {
	if err := d.checkBufferLength(Uint64Size); err != nil {
		return err
	}
	value := binary.BigEndian.Uint64(d.buffer[d.pos:])
	d.pos += Uint64Size
	v.SetUint(uint64(value))
	return nil
}

func (d *Deserializer) boolDeserializer(v reflect.Value) error {
	if err := d.checkBufferLength(1); err != nil {
		return err
	}
	value := d.buffer[d.pos] != 0
	d.pos += 1
	v.SetBool(value)
	return nil
}

func (d *Deserializer) bytesDeserializer(v reflect.Value) error {
	l, err := d.readLength()
	if err != nil {
		return err
	}
	if err := d.checkBufferLength(int(l)); err != nil {
		return err
	}
	bytes := d.buffer[d.pos : d.pos+int(l)]
	d.pos += int(l)
	v.SetBytes(bytes)
	return nil
}

func (d *Deserializer) stringDeserializer(v reflect.Value) error {
	l, err := d.readLength()
	if err != nil {
		return err
	}
	if err := d.checkBufferLength(int(l)); err != nil {
		return err
	}
	value := string(d.buffer[d.pos : d.pos+int(l)])
	d.pos += int(l)
	v.SetString(value)
	return nil
}

func (d *Deserializer) readLength() (uint64, error) {
	l, n := binary.Uvarint(d.buffer[d.pos:])
	if n <= 0 {
		return l, fmt.Errorf("can not read length")
	}
	d.pos += n
	return l, nil
}

func (d *Deserializer) checkBufferLength(l int) error {
	if len(d.buffer) - d.pos < l {
		return fmt.Errorf("exceeding buffer's length")
	}
	return nil
}

func (d *Deserializer) recursiveDeserializer(v interface{}, rv reflect.Value) error {
	switch rv.Kind() {
	case reflect.Struct:
		return d.structDeserializer(rv)
	case reflect.Array:
		return d.arrayDeserializer(rv)
	case reflect.Slice:
		return d.sliceDeserializer(rv)
	case reflect.Map:
		return d.mapDeserializer(rv)
	default:
		if d.Extension != nil {
			return d.Extension(v)
		}
		return fmt.Errorf("wrong type: %s", rv.Type().String())
	}
}

func (d *Deserializer) arrayDeserializer(rv reflect.Value) error {
	l := rv.Len()
	for i:= 0; i < l; i++ {
		if err := d.Deserialize(rv.Index(i).Addr().Interface()); err != nil {
			return err
		}
	}
	return nil
}

func (d *Deserializer) sliceDeserializer(rv reflect.Value) error {
	l, err := d.readLength()
	if err != nil {
		return err
	}
	rv.Set(reflect.MakeSlice(rv.Type(), int(l), int(l)))
	for i:= 0; i < int(l); i++ {
		if err := d.Deserialize(rv.Index(i).Addr().Interface()); err != nil {
			return err
		}
	}
	return nil
}

func (d *Deserializer) structDeserializer(rv reflect.Value) error {
	numField := rv.NumField()
	for i:= 0; i < numField; i++ {
		field := rv.Field(i)
		if field.CanSet() {
			if err := d.Deserialize(field.Addr().Interface()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *Deserializer) mapDeserializer(rv reflect.Value) error {
	l, err := d.readLength()
	if err != nil {
		return err
	}
	rv.Set(reflect.MakeMap(rv.Type()))
	for i := 0; i < int(l); i++ {
		key := reflect.Indirect(reflect.New(rv.Type().Key()))
		if err = d.Deserialize(key.Addr().Interface()); err != nil {
			return err
		}
		value := reflect.Indirect(reflect.New(rv.Type().Elem()))
		if err = d.Deserialize(value.Addr().Interface()); err != nil {
			return err
		}
		rv.SetMapIndex(key, value)
	}
	return nil
}

func UnmarshalBinary(buf []byte, v interface{}) error {
	deserializer := NewDeserializer(buf)
	return deserializer.Deserialize(v)
}