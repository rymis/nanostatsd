package metricdb

import "time"

// Quant is a timestamp in 15 second intervals from UNIX epoch
type Quant uint32

// Get current time as a quant
func QuantNow() Quant {
	return Quant(time.Now().Unix() / 15)
}

// Convert timestamp to quant
func TimeToQuant(t time.Time) Quant {
	return Quant(t.Unix() / 15)
}

// Convert quant to timestamp
func QuantStamp(q Quant) time.Time {
	return time.Unix(int64(q) * 15, 0)
}
