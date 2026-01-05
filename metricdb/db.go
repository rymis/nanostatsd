package metricdb

import (
	"sync"
	"time"
)

type MetricsDB struct {
	lock sync.Mutex


}

type GaugeResult struct {
	// Total number of captured values
	Count uint32 `json:"count"`
	// Minimal value
	Min float32 `json:"min"`
	// Maximum value
	Max float32 `json:"max"`
	// Average value
	Avg float32 `json:"avg"`
	// Average of x^2. This value can be used to calculate divirgence
	Avg2 float32 `json:"avg2"`
	// Median of the value
	P50 float32 `json:"p50"`
	// 95th percentile
	P95 float32 `json:"p95"`
	// 99th percentile
	P99 float32 `json:"p99"`

	// Histogram frame timestamp
	Timestamp float64 `json:"timestamp"`
}

type HistogramResult struct {
	// Minimal value
	Min float32 `json:"min"`
	// Maximum value
	Max float32 `json:"max"`
	// Histogram bins
	Hist []float32 `json:"hist"`
	// Histogram frame timestamp
	Timestamp float64 `json:"timestamp"`
}

type CountResult struct {
	Rps float32 `json:"rps"`
	Timestamp float64 `json:"timestamp"`
}

type MetricResult struct {
	Name string
	Tags []string
}

func (mdb *MetricsDB) WriteValue(name string, value float32, tags ...string) error {
	return nil
}

func (mdb *MetricsDB) WriteCount(name string, count float32, tags ...string) error {
	return nil
}

func (mdb *MetricsDB) WriteValueWithTime(name string, value float32, t time.Time, tags ...string) error {
	return nil
}

func (mdb *MetricsDB) WriteCountWithTime(name string, count float32, t time.Time, tags ...string) error {
	return nil
}

func (mdb *MetricsDB) QueryHistogram(name string, begin, end time.Time, tags ...string) []HistogramResult {
	return nil
}

func (mdb *MetricsDB) QueryGauge(name string, begin, end time.Time, tags ...string) []GaugeResult {
	return nil
}

func (mdb *MetricsDB) QueryCount(name string, begin, end time.Time, tags ...string) []CountResult {
	return nil
}

func (mdb *MetricsDB) ListMetrics() []MetricResult {
	return nil
}

func (mdb *MetricsDB) Close() error {
	return nil
}
