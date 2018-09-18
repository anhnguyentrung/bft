package database

import (
	"testing"
	"github.com/tecbot/gorocksdb"
	"os"
	"encoding/binary"
	"bytes"
)

var fileName = "test.db"

func TestNewRocksDB(t *testing.T) {
	rocksDB, err := setup()
	defer rocksDB.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRocksDB_AddCF(t *testing.T) {
	rocksDB, err := setup()
	defer rocksDB.Close()
	if err != nil {
		t.Fatal(err)
	}
	cfName := "blockchain"
	rocksDB.RemoveCF(cfName)
	err = rocksDB.AddCF(cfName)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rocksDB.cfHandlers[cfName]; !ok {
		t.Fatalf("column family %s is not inserted", cfName)
	}
	opts :=gorocksdb.NewDefaultOptions()
	defer opts.Destroy()
	cfNames, err := gorocksdb.ListColumnFamilies(opts, fileName)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfNames) != 2 {
		t.Fatal("wrong number of column families")
	}
	if cfNames[0] != "default" || cfNames [1] != cfName {
		t.Fatal("wrong column family name")
	}
	t.Log(cfNames)
}

func TestRocksDB_RemoveCF(t *testing.T) {
	rocksDB, err := setup()
	defer rocksDB.Close()
	if err != nil {
		t.Fatal(err)
	}
	cfName := "blockchain"
	rocksDB.AddCF(cfName)
	err = rocksDB.RemoveCF(cfName)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rocksDB.cfHandlers[cfName]; ok {
		t.Fatalf("column family %s is not deleted", cfName)
	}
	opts :=gorocksdb.NewDefaultOptions()
	defer opts.Destroy()
	cfNames, err := gorocksdb.ListColumnFamilies(opts, fileName)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfNames) != 1 {
		t.Fatal("wrong number of column families")
	}
	if cfNames[0] != "default" {
		t.Fatal("wrong column family name")
	}
	t.Log(cfNames)
}

func TestRocksDB_PutGet(t *testing.T) {
	rocksDB, _ := setup()
	defer rocksDB.Close()
	cfName := "blockchain"
	rocksDB.AddCF(cfName)
	for i := 1; i < 5; i++ {
		rocksDB.Put(cfName, encode(i), encode(i))
	}
	for i := 1; i < 5; i++ {
		result := rocksDB.Get(cfName, encode(i))
		if !bytes.Equal(result, encode(i)) {
			t.Fatal(result)
		}
	}
}

func TestRocksDB_Delete(t *testing.T) {
	rocksDB, _ := setup()
	defer rocksDB.Close()
	cfName := "blockchain"
	rocksDB.AddCF(cfName)
	rocksDB.Put(cfName, encode(1), encode(1))
	if !rocksDB.Has(cfName, encode(1)) {
		t.Fatalf("%d is not inserted", 1)
	}
	rocksDB.Delete(cfName, encode(1))
	if rocksDB.Has(cfName, encode(1)) {
		t.Fatalf("%d is not deleted", 1)
	}
}

func setup() (*RocksDB, error) {
	os.RemoveAll(fileName)
	return NewRocksDB(fileName, nil)
}

func encode(i int) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func decode(buf []byte) int {
	return int(binary.BigEndian.Uint64(buf))
}