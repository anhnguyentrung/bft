package database

import "github.com/tecbot/gorocksdb"

type BlockStore struct {
	cfName string
	db *RocksDB
}

func NewBlockStore(name string) *BlockStore {
	db := GetDB()
	db.AddCF(name)
	return &BlockStore{
		name,
		GetDB(),
	}
}

func (bs *BlockStore) Put(key, value []byte) {
	bs.db.Put(bs.cfName, key, value)
}

func (bs *BlockStore) Get(key []byte) []byte {
	return bs.db.Get(bs.cfName, key)
}

func (bs *BlockStore) Delete(key []byte) {
	bs.db.Delete(bs.cfName, key)
}

func (bs *BlockStore) Has(key []byte) bool {
	return bs.db.Has(bs.cfName, key)
}

func (bs *BlockStore) Iterator() *gorocksdb.Iterator {
	return bs.db.GetIterator(bs.cfName)
}

func (bs *BlockStore) GetFromSnapshot(snapshot *gorocksdb.Snapshot, key []byte) []byte {
	return bs.db.GetFromSnapshot(bs.cfName, snapshot, key)
}