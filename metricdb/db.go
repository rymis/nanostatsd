package metricdb

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type MetricsDB[T any] struct {
	db *sql.DB
	quantile Quant
	table string
	tx *sql.Tx
}

type MetricDBRow[T any] struct {
	Name string
	Quant Quant
	Value *T
}

func NewMetricsDB[T any](path string, quantile Quant) (*MetricsDB[T], error) {
	res := &MetricsDB[T]{}
	res.quantile = quantile
	res.table = fmt.Sprintf("metrics%d", quantile)

	dbpath := filepath.Join(path, "metrics.db")
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbpath)
	if err != nil {
		return nil, err
	}

	create := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (metric TEXT, quant INTEGER, value BLOB);", res.table)
	_, err = db.Exec(create)
	if err != nil {
		return nil, err
	}

	create = fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_metric_quant ON %s (metric, quant);", res.table, res.table)
	_, err = db.Exec(create)
	if err != nil {
		return nil, err
	}

	create = fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_quant_metric ON %s (quant, metric);", res.table, res.table)
	_, err = db.Exec(create)
	if err != nil {
		return nil, err
	}

	res.db = db

	return res, nil
}

func (mdb *MetricsDB[T]) BeginTransaction() error {
	if mdb.tx != nil {
		return errors.New("Transaction is already started")
	}

	tx, err := mdb.db.Begin()
	if err != nil {
		return err
	}

	mdb.tx = tx

	return nil
}

func (mdb *MetricsDB[T]) CommitTransaction() error {
	if mdb.tx == nil {
		return errors.New("Transaction is not started")
	}

	err := mdb.tx.Commit()
	mdb.tx = nil // If commit failed we don't want this transaction anyway

	return err
}

func (mdb *MetricsDB[T]) RollbackTransaction() error {
	if mdb.tx == nil {
		return errors.New("Transaction is not started")
	}

	err := mdb.tx.Rollback()
	mdb.tx = nil // If rollback failed we don't want this transaction anyway

	return err
}

func (mdb *MetricsDB[T]) WriteValue(name string, quant Quant, value *T) error {
	data, err := gobEncode(value)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("INSERT INTO %s VALUES (?, ?, ?);", mdb.table)

	if mdb.tx != nil {
		_, err := mdb.tx.Exec(query, name, quant, data)
		if err != nil {
			return err
		}
	} else {
		_, err := mdb.db.Exec(query, name, quant, data)
		if err != nil {
			return err
		}
	}

	return nil
}

func (mdb *MetricsDB[T]) Query(name string, begin, end Quant) ([]MetricDBRow[T], error) {
	query := fmt.Sprintf("SELECT metric, quant, value FROM %s WHERE metric == ? AND quant >= ? AND quant < ? ORDER BY metric, quant;", mdb.table)
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

	rows := make([]MetricDBRow[T], 0, 64)
	for res.Next() {
		var name string
		var quant Quant
		var data []byte

		err = res.Scan(&name, &quant, &data)
		if err != nil {
			return nil, err
		}

		val := new(T)
		err = gobDecode(data, val)
		if err != nil {
			return nil, err
		}

		rows = append(rows, MetricDBRow[T]{
			Name: name,
			Quant: quant,
			Value: val,
		})
	}

	return rows, nil
}

func (mdb *MetricsDB[T]) ListMetrics() ([]string, error) {
	query := fmt.Sprintf("SELECT DISTINCT metric FROM %s;", mdb.table)
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

func (mdb *MetricsDB[T]) Close() error {
	if mdb.tx != nil {
		mdb.tx.Rollback()
	}

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
