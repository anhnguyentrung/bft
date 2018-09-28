package types

import (
	"time"
	"bft/crypto"
	"bytes"
	"crypto/sha256"
	"log"
	"fmt"
)

type BlockHeightId struct {
	Height uint64
	Id Hash
}

func (hi BlockHeightId) IsValid() bool {
	if hi.Height == 0 {
		return false
	}
	emptyHash := Hash{}
	return !hi.Id.Equals(emptyHash)
}

func (hi BlockHeightId) Equals(target BlockHeightId) bool {
	return hi.Height == target.Height && bytes.Equal(hi.Id[:], target.Id[:])
}

func (hi BlockHeightId) String() string {
	return fmt.Sprintf("%v-%v", hi.Height, hi.Id.String())
}

type BlockHeader struct {
	HeightId BlockHeightId
	PreviousId Hash
	Proposer Validator
	Timestamp time.Time
	Commits []Vote
}

func (h BlockHeader) Height() uint64 {
	return h.HeightId.Height
}

func (h BlockHeader) Id() Hash {
	return h.HeightId.Id
}

func (h BlockHeader) CalculateId(encoder SerializeFunc) Hash {
	b, _ := encoder(h)
	return sha256.Sum256(b)
}

type SignedBlockHeader struct {
	Header BlockHeader
	Signature crypto.Signature
}

type Block struct {
	SignedHeader SignedBlockHeader
}

func NewGenesisBlock(genesis Genesis, encoder SerializeFunc) *Block {
	genesisHeader := BlockHeader{}
	genesisHeader.HeightId.Height = 1
	genesisHeader.Proposer = genesis.Proposer
	genesisHeader.Timestamp = genesis.Timestamp
	id := genesisHeader.CalculateId(encoder)
	genesisHeader.HeightId.Id = id
	signedHeader := SignedBlockHeader{
		Header: genesisHeader,
	}
	return &Block{
		signedHeader,
	}
}

func (b *Block) Header() *BlockHeader{
	return &b.SignedHeader.Header
}

func (b *Block) Height() uint64 {
	return b.Header().Height()
}

func (b *Block) Id() Hash {
	return b.Header().Id()
}

func (b *Block) Signature() crypto.Signature {
	return b.SignedHeader.Signature
}

func (b *Block) IsValid() bool {
	if b == nil {
		log.Println("block should be not nil")
		return false
	}
	if !b.Header().HeightId.IsValid() {
		log.Println("block's height or id is invalid")
		return false
	}
	if !b.Signature().IsValid() {
		log.Println("block's signature is invalid")
		return false
	}
	return true
}
