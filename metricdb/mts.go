package metricdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type MiddleTimeStorage struct {
	// Root directory
	Path string
	// Number of 15-minutes periods to store in memory (default: 1 day = 96 periods)
	Periods int

	lock sync.Mutex
	currentQuant Quant
	curFile *os.File
	metrics map[string]*TimeStorage[QuantHistogram]
	counts map[string]*TimeStorage[float32]
}

type cacheStorageValue struct {
	Name string `json:"name"`
	Quant Quant `json:"quant"`
	Histogram *QuantHistogram `json:"hist,omitempty"`
	Count float32 `json:"cnt"`
}

// Error when writing the same data again
var ErrorQuantInPast = errors.New("Quant is in past")

// Write Histogram value for metric/quant
func (mts *MiddleTimeStorage) WriteHistogram(metric string, quant Quant, data *QuantHistogram) error {
	mts.lock.Lock()
	defer mts.lock.Unlock()

	err := mts.checkQuant(quant)
	if err != nil {
		return err
	}

	// Add data to memory storage:
	ts, ok := mts.metrics[metric]
	if !ok {
		ts = NewTimeStorage[QuantHistogram]()
		mts.metrics[metric] = ts
	}

	ts.Append(quant, *data)

	// And append it to cache
	cache := &cacheStorageValue{
		Name: metric,
		Quant: quant,
		Histogram: data,
		Count: 0.0,
	}

	return mts.storeCache(cache)
}

// Write counts value for metric/quant
func (mts *MiddleTimeStorage) WriteCounts(metric string, quant Quant, rps float32) error {
	mts.lock.Lock()
	defer mts.lock.Unlock()

	err := mts.checkQuant(quant)
	if err != nil {
		return err
	}

	// Add data to memory storage:
	ts, ok := mts.counts[metric]
	if !ok {
		ts = NewTimeStorage[float32]()
		mts.counts[metric] = ts
	}

	ts.Append(quant, rps)

	// And append it to cache
	cache := &cacheStorageValue{
		Name: metric,
		Quant: quant,
		Histogram: nil,
		Count: rps,
	}

	return mts.storeCache(cache)
}

// Query metric values for time interval
func (mts *MiddleTimeStorage) QueryHistogram(metric string, begin, end Quant) []ExtendedQuantHistogram {
	mts.lock.Lock()
	defer mts.lock.Unlock()

	ts, ok := mts.metrics[metric]
	if !ok {
		return nil
	}

	data := ts.Query(begin, end)
	if len(data) == 0 {
		return nil
	}

	res := make([]ExtendedQuantHistogram, len(data))
	for i, val := range data {
		res[i].Quant = val.Quant
		res[i].QuantHistogram = val.Value
	}

	return res
}

// Query counter for time interval
func (mts *MiddleTimeStorage) QueryCounts(metric string, begin, end Quant) []QuantCount {
	mts.lock.Lock()
	defer mts.lock.Unlock()

	ts, ok := mts.counts[metric]
	if !ok {
		return nil
	}

	data := ts.Query(begin, end)
	if len(data) == 0 {
		return nil
	}

	res := make([]QuantCount, len(data))
	for i, val := range data {
		res[i].Quant = val.Quant
		res[i].Count = val.Value
	}

	return res
}

// List known metrics and counts
func (mts *MiddleTimeStorage) ListMetrics(dst map[string]bool) {
	return
}

// Call this method periodically to remove old data
func (mts *MiddleTimeStorage) Periodic() error {
	return nil
}

// Close the database
func (mts *MiddleTimeStorage) Close() error {
	err := mts.curFile.Close()
	mts.curFile = nil
	return err
}

func (mts *MiddleTimeStorage) storeCache(cache *cacheStorageValue) error {
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	_, err = mts.curFile.Write(data)
	if err != nil {
		return err
	}

	_, err = mts.curFile.WriteString("\n")
	if err != nil {
		return err
	}

	return nil
}

func (mts *MiddleTimeStorage) checkQuant(quant Quant) error {
	if quant < mts.currentQuant {
		return ErrorQuantInPast
	}

	if quant == mts.currentQuant {
		return nil
	}

	fnm := fmt.Sprintf("cache_%d.json", quant)
	path := filepath.Join(mts.Path, fnm)

	f, err := os.OpenFile(path, os.O_CREATE | os.O_APPEND | os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	err = mts.curFile.Close()
	mts.curFile = f

	return nil
}
