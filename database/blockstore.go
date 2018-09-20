package database

import (
	"github.com/tecbot/gorocksdb"
	"bft/types"
	"fmt"
	"bft/encoding"
	"log"
)

const BlockStoreCF = "blockstore"
const LastHeightKey = "lastheight"

type BlockStore struct {
	db *RocksDB
	head *types.Block
}

var blockStore = NewBlockStore()

func NewBlockStore() *BlockStore {
	db := GetDB()
	db.AddCF(BlockStoreCF)
	return &BlockStore{
		GetDB(),
		nil,
	}
}

func GetBlockStore() *BlockStore {
	return blockStore
}

func (bs *BlockStore) Head() *types.Block {
	if bs.head != nil {
		return bs.head
	}
	// try to load from database
	lastHeight := bs.LastHeight()
	if lastHeight != 0 {
		head, err := bs.GetBlockFromHeight(lastHeight)
		if err != nil {
			log.Println(err)
			return nil
		}
		bs.head = head
		return bs.head
	}
	return nil
}

func (bs *BlockStore) LastHeight() uint64 {
	if bs.head != nil {
		return bs.head.Height()
	}
	// try to load from database
	value := bs.get([]byte(LastHeightKey))
	if value == nil {
		return 0
	}
	lastHeight := uint64(0)
	encoding.UnmarshalBinary(value, &lastHeight)
	return lastHeight
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
	//save last height
	bs.saveLastHeight(height)
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

func (bs *BlockStore) saveLastHeight(height uint64) {
	value, _ := encoding.MarshalBinary(height)
	bs.put([]byte(LastHeightKey), value)
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