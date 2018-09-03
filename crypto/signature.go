package crypto

import (
	"github.com/btcsuite/btcutil/base58"
	"github.com/btcsuite/btcd/btcec"
	"bytes"
	"fmt"
)

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

func (signature *Signature) Recover(hash []byte) (*PublicKey, error) {
	recoveredKey, _, err := btcec.RecoverCompact(btcec.S256(), signature.Data, hash)
	if err != nil {
		return nil, err
	}

	return &PublicKey{Data: recoveredKey.SerializeCompressed()}, nil
}

func (signature *Signature) String() string {
	checksum := calculateCheckSum(signature.Data)
	encodeData := append(signature.Data, checksum...)
	return base58.Encode(encodeData)
}

func (signature *Signature) Verify(pubKey PublicKey, hash []byte) bool {
	recoveredPubKey, err := signature.Recover(hash)
	if err != nil {
		return false
	}
	if bytes.Equal(recoveredPubKey.Data, pubKey.Data) {
		return true
	}
	return false
}