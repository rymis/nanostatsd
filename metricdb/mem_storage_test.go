package metricdb_test

import (
	"testing"

	"github.com/rymis/nanostatsd/metricdb"
)

func TestMemStorage(t *testing.T) {
	db := metricdb.NewMemStorage[string]()

	defer func () {
		db.Close()
	}()

	testStorageImpl("MEM", db, t)
}

func TestMemStorageReduce(t *testing.T) {
	db := metricdb.NewMemStorage[string]()

	defer func () {
		db.Close()
	}()

	testStorageReduceImpl("MEM", db, t)
}

func TestMemStorageTags(t *testing.T) {
	db := metricdb.NewMemStorage[string]()

	defer func () {
		db.Close()
	}()

	testStorageTagsImpl("MEM", db, t)
}
