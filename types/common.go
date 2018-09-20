package types

import (
	"bytes"
	"bft/crypto"
	"encoding/hex"
)

type Hash [32]byte // SHA256 hash
// convert byte array to hex string
func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}
func (h Hash) Equals(target Hash) bool {
	return bytes.Equal(h[:], target[:])
}
func (h Hash) IsEmpty() bool {
	emptyHash := Hash{}
	return h.Equals(emptyHash)
}
type DeserializeFunc func (buf []byte, v interface{}) error
type SerializeFunc func (v interface{}) ([]byte, error)
type KeyPair struct {
	PrivateKey crypto.PrivateKey
	PublicKey crypto.PublicKey
}