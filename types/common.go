package types

type Hash [32]byte // SHA256 hash
type DeserializeFunc func (buf []byte, v interface{}) error