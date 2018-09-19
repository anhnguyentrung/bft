package database

import (
	"github.com/tecbot/gorocksdb"
	"bft/types"
	"fmt"
	"bft/encoding"
)

const BlockStoreCF = "blockstore"

type BlockStore struct {
	db *RocksDB
	head *types.Block
}

var blockStore = NewBlockStore()

func NewBlockStore() *BlockStore {
	if blockStore != nil {
		return blockStore
	}
	db := GetDB()
	db.AddCF(BlockStoreCF)
	blockStore = &BlockStore{
		GetDB(),
		nil,
	}
	return blockStore
}

func GetBlockStore() *BlockStore {
	return blockStore
}

func (bs *BlockStore) Head() *types.Block {
	return bs.head
}

func (bs *BlockStore) AddBlock(block *types.Block) error {
	height := block.Header().Height()
	if _, err := bs.GetBlockHeader(height); err == nil {
		return fmt.Errorf("block height %v is existing", height)
	}
	//save block header
	headerData, err := encoding.MarshalBinary(block.Header())
	if err != nil {
		return err
	}
	bs.put(keyFromHeight(height), headerData)
	//save block
	blockData, err := encoding.MarshalBinary(*block)
	if err != nil {
		return err
	}
	bs.put(keyFromId(block.Header().Id()), blockData)
	bs.head = block
	return nil
}

func (bs *BlockStore) GetBlockFromHeight(height uint64) (*types.Block, error) {
	header, err := bs.GetBlockHeader(height)
	if err != nil {
		return nil, err
	}
	return bs.GetBlockFromId(header.Id())
}

func (bs *BlockStore) GetBlockFromId(id types.Hash) (*types.Block, error) {
	key := keyFromId(id)
	value := bs.get(key)
	if value == nil {
		return nil, fmt.Errorf("block id %v does not exist", id.String())
	}
	block := types.Block{}
	encoding.UnmarshalBinary(value, &block)
	return &block, nil
}

func (bs *BlockStore) GetBlockHeader(height uint64) (*types.BlockHeader, error) {
	key := keyFromHeight(height)
	value := bs.get(key)
	if value == nil {
		return nil, fmt.Errorf("block height %v does not exist", height)
	}
	blockHeader := types.BlockHeader{}
	encoding.UnmarshalBinary(value, &blockHeader)
	return &blockHeader, nil
}

func (bs *BlockStore) RemoveBlock(height uint64) error {
	header, err := bs.GetBlockHeader(height)
	if err != nil {
		return err
	}
	//remove block header
	bs.delete(keyFromHeight(height))
	//remove block
	bs.delete(keyFromId(header.Id()))
	return nil
}

func keyFromHeight(height uint64) []byte {
	return []byte(fmt.Sprintf("H%v", height))
}

func keyFromId(id types.Hash) []byte {
	key := []byte("B")
	key = append(key, id[:]...)
	return key
}


func (bs *BlockStore) put(key, value []byte) {
	bs.db.Put(BlockStoreCF, key, value)
}

func (bs *BlockStore) get(key []byte) []byte {
	return bs.db.Get(BlockStoreCF, key)
}

func (bs *BlockStore) delete(key []byte) {
	bs.db.Delete(BlockStoreCF, key)
}

func (bs *BlockStore) has(key []byte) bool {
	return bs.db.Has(BlockStoreCF, key)
}

func (bs *BlockStore) iterator() *gorocksdb.Iterator {
	return bs.db.GetIterator(BlockStoreCF)
}

func (bs *BlockStore) getFromSnapshot(snapshot *gorocksdb.Snapshot, key []byte) []byte {
	return bs.db.GetFromSnapshot(BlockStoreCF, snapshot, key)
}