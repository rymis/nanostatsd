package metricdb

import (
	"testing"
)

func TestTagsNormalize(t *testing.T) {
	j := func (s ...string) string {
		return normalizeTags(s)
	}

	if r := j(); r != "" {
		t.Errorf("Expected '', has '%s'", r)
	}

	if r := j("t1"); r != "#t1#" {
		t.Errorf("Expected '#t1#', has '%s'", r)
	}

	if r := j("t2", "t1", "t3"); r != "#t1#t2#t3#" {
		t.Errorf("Expected '#t1#t2#t3#', has '%s'", r)
	}

	if r := j("t1", "t1", "t1"); r != "#t1#" {
		t.Errorf("Expected '#t1#', has '%s'", r)
	}

	if r := j("t1", "t2", "t1", "t3", "t2"); r != "#t1#t2#t3#" {
		t.Errorf("Expected '#t1#t2#t3#', has '%s'", r)
	}
}

func TestTagsMatch(t *testing.T) {
	j := func (nt string, s ...string) bool {
		return matchTags(s, nt)
	}

	if !j("") {
		t.Errorf("Expected match empty ''")
	}

	if j("", "t1") {
		t.Errorf("Expected not match")
	}

	if !j("#t1#t2#t3#", "t1", "t3") {
		t.Errorf("Expected match")
	}
}
