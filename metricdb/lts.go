package metricdb

import (
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
)

type LongTermStorage struct {
	db *leveldb.DB
	knownMetrics map[string]bool
	lock sync.Mutex
}
