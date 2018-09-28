package types

import (
	"time"
	"bft/crypto"
	"log"
	"crypto/sha256"
)

type Genesis struct {
	Timestamp time.Time
	Proposer Validator
}

func NewGenesis() Genesis {
	publicKey, err := crypto.NewPublicKey(GenesisProposerKey)
	if err != nil {
		log.Fatal(err)
	}
	timestamp, err := time.Parse(time.RFC3339, GenesisTime)
	if err != nil {
		log.Fatal(err)
	}
	return Genesis{
		timestamp,
		Validator{
			publicKey.Address(),
			*publicKey,
		},
	}
}

func (g Genesis) ChainId(encoder SerializeFunc) Hash {
	buf, _ := encoder(g)
	return sha256.Sum256(buf)
}
