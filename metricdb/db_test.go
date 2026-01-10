package metricdb_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/rymis/nanostatsd/metricdb"
)

func TestMetricsDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics-test")
	err := os.MkdirAll(path, 0755)
	if err != nil {
		t.Fatal(err.Error())
	}

	db, err := metricdb.NewMetricsDB[string](path)
	if err != nil {
		t.Fatal(err.Error())
	}

	defer func () {
		db.Close()
		os.RemoveAll(path)
	}()

	s := func (v string) *string {
		res := new(string)
		*res = v
		return res
	}

	jm := func (v []metricdb.MetricDBRow[string]) string {
		sort.Slice(v, func (i, j int) bool {
			return *v[i].Value < *v[j].Value
		})
		res := ""
		for i, m := range v {
			if i > 0 {
				res += "/"
			}

			res += *m.Value
		}

		return res
	}

	j := func (v []string) string {
		sort.Strings(v)
		return strings.Join(v, "/")
	}

	// Try to insert data without transaction:
	err = db.WriteValue("m1", 1, s("test-1-1"))
	if err != nil {
		t.Fatal(err.Error())
	}
	err = db.WriteValue("m1", 2, s("test-1-2"))
	if err != nil {
		t.Fatal(err.Error())
	}
	err = db.WriteValue("m2", 1, s("test-2-1"))
	if err != nil {
		t.Fatal(err.Error())
	}

	// And with transaction
	err = db.BeginTransaction()
	if err != nil {
		t.Fatal(err.Error())
	}

	err = db.WriteValue("m3", 1, s("test-3-1"))
	if err != nil {
		t.Fatal(err.Error())
	}

	err = db.CommitTransaction()
	if err != nil {
		t.Fatal(err.Error())
	}

	// And with transaction, but rollback
	err = db.BeginTransaction()
	if err != nil {
		t.Fatal(err.Error())
	}

	err = db.WriteValue("m4", 1, s("test-4-1"))
	if err != nil {
		t.Fatal(err.Error())
	}

	err = db.RollbackTransaction()
	if err != nil {
		t.Fatal(err.Error())
	}

	// Now we can read values
	mm, err := db.Query("m1", 1, 3)
	if err != nil {
		t.Fatal(err.Error())
	}

	if jm(mm) != "test-1-1/test-1-2" {
		t.Errorf("Invalid q1 result: %#v", jm(mm))
	}

	mm, err = db.Query("m1", 1, 2)
	if err != nil {
		t.Fatal(err.Error())
	}

	if jm(mm) != "test-1-1" {
		t.Errorf("Invalid q2 result: %s", jm(mm))
	}

	ss, err := db.ListMetrics()
	if err != nil {
		t.Fatal(err.Error())
	}

	if j(ss) != "m1/m2/m3" {
		t.Errorf("Invalid list of metrics: %#v", ss)
	}
}

func TestMetricsDBReduce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics-test")
	err := os.MkdirAll(path, 0755)
	if err != nil {
		t.Fatal(err.Error())
	}

	db, err := metricdb.NewMetricsDB[string](path)
	if err != nil {
		t.Fatal(err.Error())
	}

	defer func () {
		db.Close()
		os.RemoveAll(path)
	}()

	s := func (v string) *string {
		res := new(string)
		*res = v
		return res
	}

	err = db.WriteValue("m1", 3, s("test-1-3"))
	if err != nil {
		t.Fatal(err.Error())
	}
	err = db.WriteValue("m1", 4, s("test-1-4"))
	if err != nil {
		t.Fatal(err.Error())
	}
	err = db.WriteValue("m1", 5, s("test-1-5"))
	if err != nil {
		t.Fatal(err.Error())
	}
	err = db.WriteValue("m1", 6, s("test-1-6"))
	if err != nil {
		t.Fatal(err.Error())
	}
	err = db.WriteValue("m1", 8, s("test-1-8"))
	if err != nil {
		t.Fatal(err.Error())
	}
	err = db.WriteValue("m2", 8, s("test-2-8"))
	if err != nil {
		t.Fatal(err.Error())
	}

	err = db.Reduce(3, 20, func (name string, quant metricdb.Quant, bucket []string) error {
		b := strings.Join(bucket, "/")
		if name == "m1" {
			if quant == 3 {
				if b != "test-1-3/test-1-4/test-1-5" {
					t.Errorf("Invalid bucket m1-3: %s", b)
				}
			} else if quant == 6 {
				if b != "test-1-6/test-1-8" {
					t.Errorf("Invalid bucket m1-6: %s", b)
				}
			} else {
				t.Errorf("Invalid bucket: %s:%d", name, quant)
			}
		} else if name == "m2" {
			if quant == 6 {
				if b != "test-2-8" {
					t.Errorf("Invalid bucket m2-6: %s", b)
				}
			} else {
				t.Errorf("Invalid bucket: %s:%d", name, quant)
			}
		} else {
			return fmt.Errorf("Invalid metric %s", name)
		}

		return nil
	})

	db.RemoveBefore(6)
	res, err := db.Query("m1", 0, 100)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if len(res) != 2 {
		t.Errorf("Remove failed")
	}

	db.Close()
}
