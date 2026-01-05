package metricdb

// Long term data storage interface
type MetricStorage interface {
	// Write Histogram value for metric/quant
	WriteHistogram(metric string, quant Quant, data *QuantHistogram) error
	// Write counts value for metric/quant
	WriteCounts(metric string, quant Quant, rps float32) error
	// Query metric values for time interval
	QueryHistogram(metric string, begin, end Quant) []ExtendedQuantHistogram
	// Query counter for time interval
	QueryCounts(metric string, begin, end Quant) []QuantCount
	// List known metrics and counts
	ListMetrics(dst map[string]bool)
	// Call this method periodically to remove old data
	Periodic() error
	// Close database and sync all the data
	Close() error
}

type QuantCount struct {
	Quant Quant
	Count float32
}
