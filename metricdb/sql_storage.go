package metricdb

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type SqlStorage[T any] struct {
	db *sql.DB
	quantile Quant
	table string
	tx *sql.Tx
	// Prepared WriteValue queries to write values faster
	dbStmt *sql.Stmt
	txStmt *sql.Stmt
}

func NewSqlStorage[T any](path string) (*SqlStorage[T], error) {
	res := &SqlStorage[T]{}

	dbpath := filepath.Join(path, "metrics.db")
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbpath)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS metrics (metric TEXT, tags TEXT, quant INTEGER, value BLOB);")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE INDEX IF NOT EXISTS metrics_metric_quant ON metrics (metric, quant);")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE INDEX IF NOT EXISTS metrics_quant_metric ON metrics (quant, metric);")
	if err != nil {
		return nil, err
	}

	query, err := db.Prepare("INSERT INTO metrics VALUES (?, ?, ?, ?);")
	res.dbStmt = query

	res.db = db

	return res, nil
}

func (mdb *SqlStorage[T]) BeginTransaction() error {
	if mdb.tx != nil {
		return errors.New("Transaction is already started")
	}

	tx, err := mdb.db.Begin()
	if err != nil {
		return err
	}

	mdb.tx = tx
	query, err := mdb.tx.Prepare("INSERT INTO metrics VALUES (?, ?, ?, ?);")
	mdb.txStmt = query

	return nil
}

func (mdb *SqlStorage[T]) CommitTransaction() error {
	if mdb.tx == nil {
		return errors.New("Transaction is not started")
	}

	err := mdb.tx.Commit()
	mdb.tx = nil // If commit failed we don't want this transaction anyway
	mdb.txStmt.Close()
	mdb.txStmt = nil

	return err
}

func (mdb *SqlStorage[T]) RollbackTransaction() error {
	if mdb.tx == nil {
		return errors.New("Transaction is not started")
	}

	err := mdb.tx.Rollback()
	mdb.tx = nil // If rollback failed we don't want this transaction anyway
	mdb.txStmt.Close()
	mdb.txStmt = nil

	return err
}

func (mdb *SqlStorage[T]) WriteValue(name string, tags []string, quant Quant, value *T) error {
	data, err := gobEncode(value)
	if err != nil {
		return err
	}

	tt := normalizeTags(tags)

	if mdb.tx != nil {
		_, err = mdb.txStmt.Exec(name, tt, quant, data)
	} else {
		_, err = mdb.dbStmt.Exec(name, tt, quant, data)
	}
	if err != nil {
		return err
	}

	return nil
}

func (mdb *SqlStorage[T]) Query(name string, tags []string, begin, end Quant) ([]DataStorageRow[T], error) {
	query := "SELECT metric, tags, quant, value FROM metrics WHERE metric == ? AND quant >= ? AND quant < ? ORDER BY metric, quant;"
	var res *sql.Rows
	var err error

	if mdb.tx != nil {
		res, err = mdb.tx.Query(query, name, begin, end)
		if err != nil {
			return nil, err
		}
	} else {
		res, err = mdb.db.Query(query, name, begin, end)
		if err != nil {
			return nil, err
		}
	}

	defer res.Close()

	rows := make([]DataStorageRow[T], 0, 64)
	for res.Next() {
		var name string
		var quant Quant
		var data []byte
		var rowTags string

		err = res.Scan(&name, &rowTags, &quant, &data)
		if err != nil {
			return nil, err
		}

		if !matchTags(tags, rowTags) {
			continue
		}

		val := new(T)
		err = gobDecode(data, val)
		if err != nil {
			return nil, err
		}

		rows = append(rows, DataStorageRow[T]{
			Name: name,
			Quant: quant,
			Value: val,
		})
	}

	return rows, nil
}

func (mdb *SqlStorage[T]) Reduce(width, end Quant, reduce DataStorageReduce[T]) error {
	query := "SELECT metric, quant, value FROM metrics WHERE quant < ? ORDER BY metric, quant;"
	var res *sql.Rows
	var err error


	if mdb.tx != nil {
		res, err = mdb.tx.Query(query, end)
	} else {
		res, err = mdb.db.Query(query, end)
	}

	if err != nil {
		return err
	}

	defer res.Close()

	curQuant := Quant(0xffffffff)
	curName := ""
	bucket := make([]T, 0, 64)
	for res.Next() {
		var name string
		var quant Quant
		var data []byte

		err = res.Scan(&name, &quant, &data)
		if err != nil {
			return err
		}

		val := new(T)
		err = gobDecode(data, val)
		if err != nil {
			return err
		}

		if quant - quant % width != curQuant || name != curName {
			if len(bucket) > 0 {
				// TODO: tags
				err = reduce(curName, nil, curQuant, bucket)
				if err != nil {
					return err
				}
			}

			curQuant = quant - quant % width
			curName = name
			bucket = make([]T, 0, 64)
		}

		bucket = append(bucket, *val)
	}

	if len(bucket) > 0 {
		// TODO: tags
		err = reduce(curName, nil, curQuant, bucket)
		if err != nil {
			return err
		}
	}

	return nil
}

func (mdb *SqlStorage[T]) RemoveBefore(quant Quant) error {
	query := "DELETE FROM metrics WHERE quant < ?;"
	var err error

	if mdb.tx != nil {
		_, err = mdb.tx.Exec(query, quant)
	} else {
		_, err = mdb.db.Exec(query, quant)
	}

	if err != nil {
		return err
	}

	return nil
}

func (mdb *SqlStorage[T]) ListMetrics() ([]string, error) {
	query := "SELECT DISTINCT metric FROM metrics;"
	var res *sql.Rows
	var err error

	if mdb.tx != nil {
		res, err = mdb.tx.Query(query)
		if err != nil {
			return nil, err
		}
	} else {
		res, err = mdb.db.Query(query)
		if err != nil {
			return nil, err
		}
	}

	defer res.Close()

	names := make([]string, 0, 64)
	for res.Next() {
		var name string

		err = res.Scan(&name)
		if err != nil {
			return nil, err
		}

		names = append(names, name)
	}

	return names, nil
}

func (mdb *SqlStorage[T]) Close() error {
	if mdb.tx != nil {
		mdb.tx.Rollback()
	}

	mdb.dbStmt.Close()

	return mdb.db.Close()
}

func gobEncode(value any) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(value)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func gobDecode(data []byte, value any) error {
	buf := bytes.NewReader(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(value)
}
