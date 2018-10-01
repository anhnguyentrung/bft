package database

import (
	"github.com/tecbot/gorocksdb"
	"sync"
	"log"
	"fmt"
)

const DBPath  = "db2"

type RocksDB struct {
	db *gorocksdb.DB
	cfHandlers map[string]*gorocksdb.ColumnFamilyHandle
	rwMutex sync.RWMutex
}

var db = NewRocksDB(DBPath)

func NewRocksDB(path string) *RocksDB {
	rocksDB := &RocksDB{
		cfHandlers: make(map[string]*gorocksdb.ColumnFamilyHandle, 0),
	}
	err := rocksDB.open(path)
	if err != nil {
		log.Fatal(err)
	}
	return rocksDB
}

func GetDB() *RocksDB {
	return db
}

func (r *RocksDB) Close() {
	for _, chf := range r.cfHandlers {
		chf.Destroy()
	}
	r.db.Close()
	r.rwMutex.Lock()
	r.db = nil
	for cfName, _ := range r.cfHandlers {
		delete(r.cfHandlers, cfName)
	}
	r.rwMutex.Unlock()
}

func (r *RocksDB) AddCF(cfName string) error {
	if r.db == nil {
		fmt.Errorf("database should be created first\n")
	}
	opts := gorocksdb.NewDefaultOptions()
	defer  opts.Destroy()
	opts.SetCreateIfMissingColumnFamilies(true)
	opts.SetCreateIfMissing(true)
	cfh, err := r.db.CreateColumnFamily(opts, cfName)
	if err != nil {
		return err
	}
	r.rwMutex.Lock()
	defer r.rwMutex.Unlock()
	if _, ok := r.cfHandlers[cfName]; ok {
		return fmt.Errorf("column family %s is existing\n", cfName)
	}
	r.cfHandlers[cfName] = cfh
	return nil
}

func (r *RocksDB) RemoveCF(cfName string) error {
	if r.db == nil {
		return fmt.Errorf("database should be created first")
	}
	cfHandler := r.columnFamilyHandle(cfName)
	if cfHandler == nil {
		return fmt.Errorf("column family %s does not exist\n", cfName)
	}
	err := r.db.DropColumnFamily(cfHandler)
	if err != nil {
		return err
	}
	r.rwMutex.Lock()
	defer r.rwMutex.Unlock()
	delete(r.cfHandlers, cfName)
	return nil
}

func (r *RocksDB) Get(cfName string, key []byte) []byte {
	cfHandler := r.columnFamilyHandle(cfName)
	if cfHandler == nil {
		log.Fatalf("column family %s does not exist\n", cfName)
	}
	readOpt := gorocksdb.NewDefaultReadOptions()
	defer readOpt.Destroy()
	result, err := r.db.GetCF(readOpt, cfHandler, key)
	if err != nil {
		log.Fatal(err)
	}
	defer result.Free()
	if result.Data() == nil {
		return nil
	}
	data := make([]byte, result.Size())
	copy(data, result.Data())
	return data
}

func (r *RocksDB) Put(cfName string, key, value []byte) {
	cfHandler := r.columnFamilyHandle(cfName)
	if cfHandler == nil {
		log.Fatalf("column family %s does not exist\n", cfName)
	}
	writeOpt := gorocksdb.NewDefaultWriteOptions()
	defer writeOpt.Destroy()
	err := r.db.PutCF(writeOpt, cfHandler, key, value)
	if err != nil {
		log.Fatal(err)
	}
}

func (r *RocksDB) Delete(cfName string, key []byte) {
	cfHandler := r.columnFamilyHandle(cfName)
	if cfHandler == nil {
		log.Fatalf("column family %s does not exist\n", cfName)
	}
	writeOpt := gorocksdb.NewDefaultWriteOptions()
	defer writeOpt.Destroy()
	err := r.db.DeleteCF(writeOpt, cfHandler, key)
	if err != nil {
		log.Fatal(err)
	}
}

func (r *RocksDB) Has(cfName string, key []byte) bool {
	return r.Get(cfName, key) != nil
}

func (r *RocksDB) GetFromSnapshot(cfName string, snapshot *gorocksdb.Snapshot, key []byte) []byte {
	cfHandler := r.columnFamilyHandle(cfName)
	if cfHandler == nil {
		log.Fatalf("column family %s does not exist\n", cfName)
	}
	readOpt := gorocksdb.NewDefaultReadOptions()
	defer readOpt.Destroy()
	readOpt.SetSnapshot(snapshot)
	result, err := r.db.GetCF(readOpt, cfHandler, key)
	if err != nil {
		log.Fatal(err)
	}
	defer result.Free()
	if result.Data() == nil {
		return nil
	}
	data := make([]byte, 0)
	copy(data, result.Data())
	return data
}

func (r *RocksDB) GetIterator(cfName string) *gorocksdb.Iterator {
	cfHandler := r.columnFamilyHandle(cfName)
	if cfHandler == nil {
		log.Fatalf("column family %s does not exist\n", cfName)
	}
	readOpt := gorocksdb.NewDefaultReadOptions()
	readOpt.SetFillCache(true)
	defer readOpt.Destroy()
	return r.db.NewIteratorCF(readOpt, cfHandler)
}

func (r *RocksDB) GetSnapshotIterator(cfName string, snapshot *gorocksdb.Snapshot) *gorocksdb.Iterator {
	cfHandler := r.columnFamilyHandle(cfName)
	if cfHandler == nil {
		log.Fatalf("column family %s does not exist\n", cfName)
	}
	readOpt := gorocksdb.NewDefaultReadOptions()
	defer readOpt.Destroy()
	readOpt.SetSnapshot(snapshot)
	return r.db.NewIteratorCF(readOpt, cfHandler)
}

func (r *RocksDB) open(path string) error {
	opts := gorocksdb.NewDefaultOptions()
	defer  opts.Destroy()
	opts.SetCreateIfMissingColumnFamilies(true)
	opts.SetCreateIfMissing(true)
	cfNames := []string{"default"}
	existedCFNames, _ := gorocksdb.ListColumnFamilies(opts, path)
	cfNames = append(cfNames, existedCFNames...)
	cfOpts := make([]*gorocksdb.Options, 0)
	for range cfNames {
		cfOpts = append(cfOpts, opts)
	}
	db, cfhs, err := gorocksdb.OpenDbColumnFamilies(opts, path, cfNames, cfOpts)
	if err != nil {
		return err
	}
	r.rwMutex.Lock()
	r.db = db
	for i, cfh := range cfhs {
		// ignore default column family
		if i > 0 {
			r.cfHandlers[cfNames[i]] = cfh
		}
	}
	r.rwMutex.Unlock()
	return nil
}

func (r *RocksDB) columnFamilyHandle(cfName string) *gorocksdb.ColumnFamilyHandle {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()
	return r.cfHandlers[cfName]
}
