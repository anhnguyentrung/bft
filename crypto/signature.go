package crypto

import (
	"github.com/btcsuite/btcutil/base58"
	"github.com/btcsuite/btcd/btcec"
	"bytes"
	"fmt"
)

type SignFunc func (digest []byte) (Signature, error)

type Signature struct {
	Data []byte
}

func NewSignature(sigString string) (*Signature, error) {
	decode := base58.Decode(sigString)
	data := decode[:len(decode)-4]
	checksum := decode[len(decode)-4:]
	if !bytes.Equal(calculateCheckSum(data), checksum) {
		return nil, fmt.Errorf("invalid checksum")
	}
	return &Signature{Data: data}, nil
}

func (s *Signature) Recover(hash []byte) (*PublicKey, error) {
	recoveredKey, _, err := btcec.RecoverCompact(btcec.S256(), s.Data, hash)
	if err != nil {
		return nil, err
	}

	return &PublicKey{Data: recoveredKey.SerializeCompressed()}, nil
}

func (s *Signature) String() string {
	checksum := calculateCheckSum(s.Data)
	encodeData := append(s.Data, checksum...)
	return base58.Encode(encodeData)
}

func (s *Signature) Verify(address string, hash []byte) bool {
	recoveredPubKey, err := s.Recover(hash)
	if err != nil {
		return false
	}
	if recoveredPubKey.Address() == address {
		return true
	}
	return false
}

func (s Signature) IsValid() bool {
	emptySig := make([]byte, 65, 65)
	return !bytes.Equal(s.Data, emptySig)
}