package metricdb

import (
	"errors"
)

type MemStorage[T any] struct {
	quantile Quant
	table string

	data map[string]*TimeStorage[T]
	tx map[string]*TimeStorage[T]
}

func NewMemStorage[T any]() *MemStorage[T] {
	res := &MemStorage[T]{}

	res.data = make(map[string]*TimeStorage[T])

	return res
}

func (mdb *MemStorage[T]) BeginTransaction() error {
	if mdb.tx != nil {
		return errors.New("Transaction is already started")
	}

	mdb.tx = make(map[string]*TimeStorage[T])

	return nil
}

func (mdb *MemStorage[T]) CommitTransaction() error {
	if mdb.tx == nil {
		return errors.New("Transaction is not started")
	}

	for nm, ts := range mdb.tx {
		ts0, ok := mdb.data[nm]
		if ok {
			ts0.Merge(ts)
		} else {
			mdb.data[nm] = ts
		}
	}

	mdb.tx = nil

	return nil
}

func (mdb *MemStorage[T]) RollbackTransaction() error {
	if mdb.tx == nil {
		return errors.New("Transaction is not started")
	}

	mdb.tx = nil

	return nil
}

func (mdb *MemStorage[T]) WriteValue(name string, quant Quant, value *T) error {
	var ts *TimeStorage[T]
	var ok bool

	if mdb.tx != nil {
		ts, ok = mdb.tx[name]
		if !ok {
			ts = NewTimeStorage[T]()
			mdb.tx[name] = ts
		}
	} else {
		ts, ok = mdb.data[name]
		if !ok {
			ts = NewTimeStorage[T]()
			mdb.data[name] = ts
		}
	}

	ts.Append(quant, *value)

	return nil
}

func (mdb *MemStorage[T]) Query(name string, begin, end Quant) ([]DataStorageRow[T], error) {
	rows := make([]DataStorageRow[T], 0, 64)
	if mdb.tx != nil {
		ts, ok := mdb.tx[name]
		if ok {
			for _, el := range ts.Query(begin, end) {
				rows = append(rows, DataStorageRow[T]{
					Name: name,
					Quant: el.Quant,
					Value: &el.Value,
				})
			}
		}

	}

	ts, ok := mdb.data[name]
	if ok {
		for _, el := range ts.Query(begin, end) {
			rows = append(rows, DataStorageRow[T]{
				Name: name,
				Quant: el.Quant,
				Value: &el.Value,
			})
		}
	}

	return rows, nil
}

func (mdb *MemStorage[T]) Reduce(width, end Quant, reduce func (name string, quant Quant, bucket []T) error) error {
	// In this implementation I ignore transactions
	for name, ts := range mdb.data {
		if len(ts.Values) == 0 {
			continue
		}

		minQuant := ts.Values[0].Quant - ts.Values[0].Quant % width
		for q := minQuant; q < end; q += width {
			eq := q + width
			if eq > end {
				eq = end
			}
			elements := ts.Query(q, eq)
			if len(elements) == 0 {
				continue
			}
			buck := make([]T, len(elements))

			for i := range elements {
				buck[i] = elements[i].Value
			}

			err := reduce(name, q, buck)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (mdb *MemStorage[T]) RemoveBefore(quant Quant) error {
	for _, ts := range mdb.data {
		ts.RemoveBefore(quant)
	}

	return nil
}

func (mdb *MemStorage[T]) ListMetrics() ([]string, error) {
	res := make([]string, 0, len(mdb.data))

	for name := range mdb.data {
		res = append(res, name)
	}

	return res, nil
}

func (mdb *MemStorage[T]) Close() error {
	return nil
}
