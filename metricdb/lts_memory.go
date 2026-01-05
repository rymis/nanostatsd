package metricdb

// Long term data storage interface
type LongTermMemoryStorage struct {

}

// Write Histogram value for metric/quant
func (ms *LongTermMemoryStorage) WriteHistogram(metric string, quant Quant, data *QuantHistogram) {

}

// Write counts value for metric/quant
func (ms *LongTermMemoryStorage) WriteCounts(metric string, quant Quant, rps float32) {

}

// Query metric values for time interval
func (ms *LongTermMemoryStorage) QueryHistogram(metric string, begin, end Quant) []QuantHistogram {
	return nil
}

// Query counter for time interval
func (ms *LongTermMemoryStorage) QueryCounts(metric string, begin, end Quant) []float32 {
	return nil
}

// List known metrics and counts
func (ms *LongTermMemoryStorage) ListMetrics() []string {
	return nil
}

// Call this method periodically to remove old data
func (ms *LongTermMemoryStorage) Periodic() {
}
