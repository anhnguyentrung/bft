package database

import (
	"github.com/tecbot/gorocksdb"
	"sync"
	"log"
	"fmt"
)

const defaultPath  = "db2"

type RocksDB struct {
	db *gorocksdb.DB
	cfHandlers map[string]*gorocksdb.ColumnFamilyHandle
	rwMutex sync.RWMutex
}

var db = NewRocksDB(defaultPath)

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

func (rocksDB *RocksDB) Close() {
	for _, chf := range rocksDB.cfHandlers {
		chf.Destroy()
	}
	rocksDB.db.Close()
	rocksDB.rwMutex.Lock()
	rocksDB.db = nil
	for cfName, _ := range rocksDB.cfHandlers {
		delete(rocksDB.cfHandlers, cfName)
	}
	rocksDB.rwMutex.Unlock()
}

func (rocksDB *RocksDB) AddCF(cfName string) error {
	if rocksDB.db == nil {
		fmt.Errorf("database should be created first\n")
	}
	opts := gorocksdb.NewDefaultOptions()
	defer  opts.Destroy()
	opts.SetCreateIfMissingColumnFamilies(true)
	opts.SetCreateIfMissing(true)
	cfh, err := rocksDB.db.CreateColumnFamily(opts, cfName)
	if err != nil {
		return err
	}
	rocksDB.rwMutex.Lock()
	defer rocksDB.rwMutex.Unlock()
	if _, ok := rocksDB.cfHandlers[cfName]; ok {
		return fmt.Errorf("column family %s is existing\n", cfName)
	}
	rocksDB.cfHandlers[cfName] = cfh
	return nil
}

func (rocksDB *RocksDB) RemoveCF(cfName string) error {
	if rocksDB.db == nil {
		return fmt.Errorf("database should be created first")
	}
	cfHandler := rocksDB.columnFamilyHandle(cfName)
	if cfHandler == nil {
		return fmt.Errorf("column family %s does not exist\n", cfName)
	}
	err := rocksDB.db.DropColumnFamily(cfHandler)
	if err != nil {
		return err
	}
	rocksDB.rwMutex.Lock()
	defer rocksDB.rwMutex.Unlock()
	delete(rocksDB.cfHandlers, cfName)
	return nil
}

func (rocksDB *RocksDB) Get(cfName string, key []byte) []byte {
	cfHandler := rocksDB.columnFamilyHandle(cfName)
	if cfHandler == nil {
		log.Fatalf("column family %s does not exist\n", cfName)
	}
	readOpt := gorocksdb.NewDefaultReadOptions()
	defer readOpt.Destroy()
	result, err := rocksDB.db.GetCF(readOpt, cfHandler, key)
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

func (rocksDB *RocksDB) Put(cfName string, key, value []byte) {
	cfHandler := rocksDB.columnFamilyHandle(cfName)
	if cfHandler == nil {
		log.Fatalf("column family %s does not exist\n", cfName)
	}
	writeOpt := gorocksdb.NewDefaultWriteOptions()
	defer writeOpt.Destroy()
	err := rocksDB.db.PutCF(writeOpt, cfHandler, key, value)
	if err != nil {
		log.Fatal(err)
	}
}

func (rocksDB *RocksDB) Delete(cfName string, key []byte) {
	cfHandler := rocksDB.columnFamilyHandle(cfName)
	if cfHandler == nil {
		log.Fatalf("column family %s does not exist\n", cfName)
	}
	writeOpt := gorocksdb.NewDefaultWriteOptions()
	defer writeOpt.Destroy()
	err := rocksDB.db.DeleteCF(writeOpt, cfHandler, key)
	if err != nil {
		log.Fatal(err)
	}
}

func (rocksDB *RocksDB) Has(cfName string, key []byte) bool {
	return rocksDB.Get(cfName, key) != nil
}

func (rocksDB *RocksDB) GetFromSnapshot(cfName string, snapshot *gorocksdb.Snapshot, key []byte) []byte {
	cfHandler := rocksDB.columnFamilyHandle(cfName)
	if cfHandler == nil {
		log.Fatalf("column family %s does not exist\n", cfName)
	}
	readOpt := gorocksdb.NewDefaultReadOptions()
	defer readOpt.Destroy()
	readOpt.SetSnapshot(snapshot)
	result, err := rocksDB.db.GetCF(readOpt, cfHandler, key)
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

func (rocksDB *RocksDB) GetIterator(cfName string) *gorocksdb.Iterator {
	cfHandler := rocksDB.columnFamilyHandle(cfName)
	if cfHandler == nil {
		log.Fatalf("column family %s does not exist\n", cfName)
	}
	readOpt := gorocksdb.NewDefaultReadOptions()
	readOpt.SetFillCache(true)
	defer readOpt.Destroy()
	return rocksDB.db.NewIteratorCF(readOpt, cfHandler)
}

func (rocksDB *RocksDB) GetSnapshotIterator(cfName string, snapshot *gorocksdb.Snapshot) *gorocksdb.Iterator {
	cfHandler := rocksDB.columnFamilyHandle(cfName)
	if cfHandler == nil {
		log.Fatalf("column family %s does not exist\n", cfName)
	}
	readOpt := gorocksdb.NewDefaultReadOptions()
	defer readOpt.Destroy()
	readOpt.SetSnapshot(snapshot)
	return rocksDB.db.NewIteratorCF(readOpt, cfHandler)
}

func (rocksDB *RocksDB) open(path string) error {
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
	rocksDB.rwMutex.Lock()
	rocksDB.db = db
	for i, cfh := range cfhs {
		// ignore default column family
		if i > 0 {
			rocksDB.cfHandlers[cfNames[i]] = cfh
		}
	}
	rocksDB.rwMutex.Unlock()
	return nil
}

func (rocksDB *RocksDB) columnFamilyHandle(cfName string) *gorocksdb.ColumnFamilyHandle {
	rocksDB.rwMutex.RLock()
	defer rocksDB.rwMutex.RUnlock()
	return rocksDB.cfHandlers[cfName]
}
