package types

import (
	"bytes"
	"bft/crypto"
)

type Hash [32]byte // SHA256 hash
func (h Hash) Equals(target Hash) bool {
	return bytes.Equal(h[:], target[:])
}
type DeserializeFunc func (buf []byte, v interface{}) error
type SerializeFunc func (v interface{}) ([]byte, error)
type EnDecoder struct {
	Encoder SerializeFunc
	Decoder DeserializeFunc
}
type KeyPair struct {
	PrivateKey crypto.PrivateKey
	PublicKey crypto.PublicKey
}
type Signfunc func (digest []byte) (crypto.Signature, error)