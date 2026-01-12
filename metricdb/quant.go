package metricdb

import (
	"encoding/json"
	"time"
)

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

func (q Quant) MarshalJSON() ([]byte, error) {
	v := float64(q) * 15.0
	return json.Marshal(v)
}

func (q *Quant) UnmarshalJSON(data []byte) error {
	var v float64
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}

	*q = Quant(v / 15.0)

	return nil
}
