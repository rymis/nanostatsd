package metricdb_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rymis/nanostatsd/metricdb"
)

func TestSqlStorage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics-test")
	err := os.MkdirAll(path, 0755)
	if err != nil {
		t.Fatal(err.Error())
	}

	db, err := metricdb.NewSqlStorage[string](path)
	if err != nil {
		t.Fatal(err.Error())
	}

	defer func () {
		db.Close()
		os.RemoveAll(path)
	}()

	testStorageImpl("SQL", db, t)
}

func TestSqlStorageReduce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics-test")
	err := os.MkdirAll(path, 0755)
	if err != nil {
		t.Fatal(err.Error())
	}

	db, err := metricdb.NewSqlStorage[string](path)
	if err != nil {
		t.Fatal(err.Error())
	}

	defer func () {
		db.Close()
		os.RemoveAll(path)
	}()

	testStorageReduceImpl("SQL", db, t)
}

func TestSqlStorageTags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics-test")
	err := os.MkdirAll(path, 0755)
	if err != nil {
		t.Fatal(err.Error())
	}

	db, err := metricdb.NewSqlStorage[string](path)
	if err != nil {
		t.Fatal(err.Error())
	}

	defer func () {
		db.Close()
		os.RemoveAll(path)
	}()

	testStorageTagsImpl("SQL", db, t)
}
