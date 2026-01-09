package metricdb_test

import (
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

	db, err := metricdb.NewMetricsDB[string](path, 15)
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
