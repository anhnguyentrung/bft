package crypto

import (
	"github.com/btcsuite/btcutil/base58"
	"fmt"
	"bytes"
	"golang.org/x/crypto/ripemd160"
)

type PublicKey struct {
	Data []byte
}

func NewPublicKey(pubString string) (*PublicKey, error) {
	decode := base58.Decode(pubString)
	checkSum := make([]byte, 4)
	copy(checkSum, decode[len(decode)-4:])
	data := decode[:len(decode)-4]
	if !bytes.Equal(calculateCheckSum(data), checkSum) {
		return nil, fmt.Errorf("invalid checksum")
	}
	return &PublicKey{data}, nil
}

func calculateCheckSum(input []byte) []byte {
	r160 := ripemd160.New()
	sum := r160.Sum(input)
	return sum[:4]
}

func (pk *PublicKey) String() string {
	checkSum := calculateCheckSum(pk.Data)
	encodeData := append(pk.Data, checkSum...)
	return base58.Encode(encodeData)
}

func (pk *PublicKey) Address() string {
	return pk.String()
}

func (pk PublicKey) Equals(target PublicKey) bool {
	return bytes.Equal(pk.Data, target.Data)
}
