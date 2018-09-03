package crypto

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"fmt"
	"io"
	"crypto/sha256"
	"crypto/rand"
	"github.com/btcsuite/btcutil"
)

type PrivateKey struct {
	*btcec.PrivateKey
}

func NewRandomPrivateKey() (*PrivateKey, error) {
	randomBytes := make([]byte, 32)
	n, err := io.ReadFull(rand.Reader, randomBytes)
	if err!= nil {
		return nil, err
	}
	if n != 32 {
		return nil, fmt.Errorf("wrong length")
	}
	hash := sha256.Sum256(randomBytes)
	privateKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), hash[:])
	return &PrivateKey{privateKey}, nil
}

func NewPrivateKey(wifString string) (*PrivateKey, error) {
	wif, err := btcutil.DecodeWIF(wifString)
	if err != nil {
		return nil, err
	}
	return &PrivateKey{wif.PrivKey}, nil
}

func (privateKey *PrivateKey) PublicKey() *PublicKey {
	pubData := privateKey.PubKey().SerializeCompressed()
	return &PublicKey{Data: pubData}
}

func (privateKey *PrivateKey) Sign(hash []byte) (Signature, error) {
	if len(hash) != 32 {
		return Signature{}, fmt.Errorf("wrong length")
	}
	sigData, err :=  btcec.SignCompact(btcec.S256(), privateKey.PrivateKey, hash, true)
	if err != nil {
		return Signature{}, err
	}
	return Signature{Data: sigData}, nil
}

func (privateKey *PrivateKey) String() string {
	wif, _ := btcutil.NewWIF(privateKey.PrivateKey, &chaincfg.Params{PrivateKeyID:'\x80'}, false)
	return wif.String()
}
