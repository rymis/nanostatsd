package metricdb_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/rymis/nanostatsd/metricdb"
)

func testStorageImpl(name string, db metricdb.DataStorage[string], t *testing.T) {
	s := func (v string) *string {
		res := new(string)
		*res = v
		return res
	}

	jm := func (v []metricdb.DataStorageRow[string]) string {
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
	err := db.WriteValue("m1", nil, 1, s("test-1-1"))
	if err != nil {
		t.Fatalf("[%s]: %s", name, err.Error())
	}
	err = db.WriteValue("m1", nil, 2, s("test-1-2"))
	if err != nil {
		t.Fatalf("[%s]: %s", name, err.Error())
	}
	err = db.WriteValue("m2", nil, 1, s("test-2-1"))
	if err != nil {
		t.Fatalf("[%s]: %s", name, err.Error())
	}

	// And with transaction
	err = db.BeginTransaction()
	if err != nil {
		t.Fatalf("[%s]: %s", name, err.Error())
	}

	err = db.WriteValue("m3", nil, 1, s("test-3-1"))
	if err != nil {
		t.Fatalf("[%s]: %s", name, err.Error())
	}

	err = db.CommitTransaction()
	if err != nil {
		t.Fatalf("[%s]: %s", name, err.Error())
	}

	// And with transaction, but rollback
	err = db.BeginTransaction()
	if err != nil {
		t.Fatalf("[%s]: %s", name, err.Error())
	}

	err = db.WriteValue("m4", nil, 1, s("test-4-1"))
	if err != nil {
		t.Fatalf("[%s]: %s", name, err.Error())
	}

	err = db.RollbackTransaction()
	if err != nil {
		t.Fatalf("[%s]: %s", name, err.Error())
	}

	// Now we can read values
	mm, err := db.Query("m1", nil, 1, 3)
	if err != nil {
		t.Fatalf("[%s]: %s", name, err.Error())
	}

	if jm(mm) != "test-1-1/test-1-2" {
		t.Errorf("[%s]: Invalid q1 result: %#v", name, jm(mm))
	}

	mm, err = db.Query("m1", nil, 1, 2)
	if err != nil {
		t.Fatalf("[%s]: %s", name, err.Error())
	}

	if jm(mm) != "test-1-1" {
		t.Errorf("[%s]: Invalid q2 result: %s", name, jm(mm))
	}

	ss, err := db.ListMetrics()
	if err != nil {
		t.Fatalf("[%s]: %s", name, err.Error())
	}

	if j(ss) != "m1/m2/m3" {
		t.Errorf("[%s]: Invalid list of metrics: %#v", name, ss)
	}
}

func testStorageReduceImpl(testName string, db metricdb.DataStorage[string], t *testing.T) {
	s := func (v string) *string {
		res := new(string)
		*res = v
		return res
	}

	err := db.WriteValue("m1", nil, 3, s("test-1-3"))
	if err != nil {
		t.Fatalf("[%s]: %s", testName, err.Error())
	}
	err = db.WriteValue("m1", nil, 4, s("test-1-4"))
	if err != nil {
		t.Fatalf("[%s]: %s", testName, err.Error())
	}
	err = db.WriteValue("m1", nil, 5, s("test-1-5"))
	if err != nil {
		t.Fatalf("[%s]: %s", testName, err.Error())
	}
	err = db.WriteValue("m1", nil, 6, s("test-1-6"))
	if err != nil {
		t.Fatalf("[%s]: %s", testName, err.Error())
	}
	err = db.WriteValue("m1", nil, 8, s("test-1-8"))
	if err != nil {
		t.Fatalf("[%s]: %s", testName, err.Error())
	}
	err = db.WriteValue("m2", nil, 8, s("test-2-8"))
	if err != nil {
		t.Fatalf("[%s]: %s", testName, err.Error())
	}

	err = db.Reduce(3, 20, func (name string, tags []string, quant metricdb.Quant, bucket []string) error {
		b := strings.Join(bucket, "/")
		if name == "m1" {
			if quant == 3 {
				if b != "test-1-3/test-1-4/test-1-5" {
					t.Errorf("[%s]: Invalid bucket m1-3: %s", testName, b)
				}
			} else if quant == 6 {
				if b != "test-1-6/test-1-8" {
					t.Errorf("[%s]: Invalid bucket m1-6: %s", testName, b)
				}
			} else {
				t.Errorf("[%s]: Invalid bucket: %s:%d", testName, name, quant)
			}
		} else if name == "m2" {
			if quant == 6 {
				if b != "test-2-8" {
					t.Errorf("[%s]: Invalid bucket m2-6: %s", testName, b)
				}
			} else {
				t.Errorf("[%s]: Invalid bucket: %s:%d", testName, name, quant)
			}
		} else {
			return fmt.Errorf("[%s]: Invalid metric %s", testName, name)
		}

		return nil
	})

	db.RemoveBefore(6)
	res, err := db.Query("m1", nil, 0, 100)
	if err != nil {
		t.Fatalf("[%s]: Error: %v", testName, err)
	}

	if len(res) != 2 {
		t.Errorf("[%s]: Remove failed", testName)
	}
}

func testStorageTagsImpl(testName string, db metricdb.DataStorage[string], t *testing.T) {
	s := func (v string) *string {
		res := new(string)
		*res = v
		return res
	}

	tags := func (t ...string) []string {
		return t
	}

	err := db.WriteValue("m1", tags("a"), 1, s("ta"))
	if err != nil {
		t.Fatalf("[%s]: %s", testName, err.Error())
	}
	err = db.WriteValue("m1", tags("b"), 1, s("tb"))
	if err != nil {
		t.Fatalf("[%s]: %s", testName, err.Error())
	}
	err = db.WriteValue("m1", tags("a", "b"), 1, s("tab"))
	if err != nil {
		t.Fatalf("[%s]: %s", testName, err.Error())
	}

	err = db.Reduce(3, 100, func (name string, tags []string, quant metricdb.Quant, bucket []string) error {
		b := strings.Join(bucket, "/")
		ts := strings.Join(tags, "#")
		if name == "m1" {
			if ts == "a" {
				if b != "ta" {
					t.Errorf("[%s]: Invalid bucket a: %s", testName, b)
				}
			} else if ts == "b" {
				if b != "tb" {
					t.Errorf("[%s]: Invalid bucket b: %s", testName, b)
				}
			} else if ts == "a#b" {
				if b != "tab" {
					t.Errorf("[%s]: Invalid bucket ab: %s", testName, b)
				}
			} else {
				t.Errorf("[%s]: Invalid bucket: %s:%d", testName, name, quant)
			}
		} else {
			return fmt.Errorf("[%s]: Invalid metric %s", testName, name)
		}

		return nil
	})
}
