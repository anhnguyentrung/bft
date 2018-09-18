package database

import (
	"github.com/tecbot/gorocksdb"
	"sync"
	"log"
)

type RocksDB struct {
	db *gorocksdb.DB
	cfHandlers map[string]*gorocksdb.ColumnFamilyHandle
	rwMutex sync.RWMutex
}

func NewRocksDB(path string, cfNames []string) *RocksDB {
	rocksDB := &RocksDB{
		cfHandlers: make(map[string]*gorocksdb.ColumnFamilyHandle, 0),
	}
	err := rocksDB.open(path, cfNames)
	if err != nil {
		log.Fatal(err)
	}
	return rocksDB
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

func (rocksDB *RocksDB) AddCF(cfName string, force bool) {
	if rocksDB.db == nil {
		log.Println("database should be created first")
	}
	opts := gorocksdb.NewDefaultOptions()
	defer  opts.Destroy()
	opts.SetCreateIfMissingColumnFamilies(true)
	opts.SetCreateIfMissing(true)
	cfh, err := rocksDB.db.CreateColumnFamily(opts, cfName)
	if err != nil {
		log.Fatal(err)
	}
	rocksDB.rwMutex.Lock()
	defer rocksDB.rwMutex.Unlock()
	if _, ok := rocksDB.cfHandlers[cfName]; ok {
		if !force {
			log.Printf("column family %s is existing\n", cfName)
			return
		}
	}
	rocksDB.cfHandlers[cfName] = cfh
}

func (rocksDB *RocksDB) RemoveCF(cfName string) {
	if rocksDB.db == nil {
		log.Println("database should be created first")
	}
	cfHandler := rocksDB.columnFamilyHandle(cfName)
	if cfHandler == nil {
		log.Fatalf("column family %s does not exist\n", cfName)
	}
	err := rocksDB.db.DropColumnFamily(cfHandler)
	if err != nil {
		log.Fatal(err)
	}
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
	data := make([]byte, 0)
	copy(data, result.Data())
	return data
}

func (rocksDB *RocksDB) Put(cfName string, key []byte, value []byte) {
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

func (rocksDB *RocksDB) open(path string, cfNames []string) error {
	opts := gorocksdb.NewDefaultOptions()
	defer  opts.Destroy()
	opts.SetCreateIfMissingColumnFamilies(true)
	opts.SetCreateIfMissing(true)
	givenCFNames := []string{"default"}
	givenCFNames = append(givenCFNames, cfNames...)
	cfOpts := make([]*gorocksdb.Options, 0)
	for range givenCFNames {
		cfOpts = append(cfOpts, opts)
	}
	db, cfhs, err := gorocksdb.OpenDbColumnFamilies(opts, path, givenCFNames, cfOpts)
	if err != nil {
		return err
	}
	rocksDB.rwMutex.Lock()
	rocksDB.db = db
	for i, cfh := range cfhs {
		// ignore default column family
		if i > 0 {
			rocksDB.cfHandlers[givenCFNames[i]] = cfh
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
