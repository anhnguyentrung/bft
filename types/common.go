package types

import "bytes"

type Hash [32]byte // SHA256 hash
func (h Hash) Equals(target Hash) bool {
	return bytes.Equal(h[:], target[:])
}
type DeserializeFunc func (buf []byte, v interface{}) error
type SerializeFunc func (v interface{}) ([]byte, error)