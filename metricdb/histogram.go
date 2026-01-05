package metricdb

import (
	"bytes"
	"encoding/gob"
	"math"
	"sort"
)

// Histogram and gauge value for one quant of time
type QuantHistogram struct {
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
	// QuantHistogram heights
	Hist []float32 `json:"hist"`
}

// Extended quant histogram contains histogram and key
type ExtendedQuantHistogram struct {
	QuantHistogram
	// Time quant of this histogram
	Quant Quant
}

// Create new quant histogram from collected values
func NewQuantHistogram(values []float32) *QuantHistogram {
	var res QuantHistogram
	if len(values) == 0 {
		return &res
	}

	res.Count = uint32(len(values))
	res.Min = values[0]
	res.Max = values[0]

	avg := float64(0.0)
	avg2 := float64(0.0)
	for _, x := range values {
		avg += float64(x)
		avg2 += float64(x * x)
		if x < res.Min {
			res.Min = x
		}
		if x > res.Max {
			res.Max = x
		}
	}
	res.Avg = float32(avg / float64(len(values)))
	res.Avg2 = float32(avg2 / float64(len(values)))

	// For small number of values we don't have enough information, so we'd better put the values themselve
	if len(values) < 16 {
		res.Hist = append(res.Hist, values...)
		return &res
	}

	// Prepare histogram and average
	// Let's use Terrell–Scott rule to get number of bins:
	cnt := binCountTerrellScott(len(values))

	delta := res.Max - res.Min
	if delta < math.SmallestNonzeroFloat32 {
		delta = 1.0
	}
	res.Hist = make([]float32, cnt)

	for _, f := range values {
		if cnt == 1 {
			res.Hist[0] += 1.0
		} else {
			i := int((f - res.Min) * float32(cnt) / delta) 
			if i >= len(res.Hist) {
				i = len(res.Hist) - 1
			}
			res.Hist[i] += 1.0
		}
	}

	return &res
}

// Get actual histogram to display
func (h *QuantHistogram) Histogram() (left, right float32, bins []uint32) {
	left = h.Min
	right = h.Max
	bins = nil
	if int(h.Count) == len(h.Hist) { // All the values are in Hist
		if len(h.Hist) == 0 {
			return
		}

		cnt := binCountTerrellScott(len(h.Hist))
		d := (right - left) / float32(cnt)
		if d < math.SmallestNonzeroFloat32 {
			d = 1.0
		}
		bins = make([]uint32, cnt)
		for _, x := range h.Hist {
			i := int((x - left) / d)
			if i >= cnt {
				i = cnt - 1
			}
			bins[i]++
		}

		return
	}

	bins = make([]uint32, len(h.Hist))
	for i := 0; i < len(bins); i++ {
		bins[i] = uint32(h.Hist[i])
	}

	return
}

// Merge several quants of histograms into one.
// The final value is not precise, but it allows to save a lot of space.
func MergeQuantHistograms(metrics []QuantHistogram) QuantHistogram {
	var res QuantHistogram
	avg := float64(0.0)
	avg2 := float64(0.0)
	res.Min = metrics[0].Min
	res.Max = metrics[0].Max

	for _, m := range metrics {
		if m.Count > 0 {
			avg += float64(m.Avg) * float64(m.Count)
			avg2 += float64(m.Avg2) * float64(m.Count)
			res.Count += m.Count

			if m.Min < res.Min {
				res.Min = m.Min
			}

			if m.Max > res.Max {
				res.Max = m.Max
			}
		}
	}

	res.Avg = float32(avg / float64(res.Count))
	res.Avg2 = float32(avg2 / float64(res.Count))

	cnt := binCountTerrellScott(int(res.Count))

	// OK, let's prepare histogram
	res.Hist = make([]float32, cnt)
	delta := res.Max - res.Min
	if delta < math.SmallestNonzeroFloat32 {
		delta = 1.0
	}

	for i := 0; i < cnt; i++ {
		l := res.Min + float32(i) * delta / float32(cnt)
		r := res.Min + float32(i + 1) * delta / float32(cnt)
		for j := 0; j < len(metrics); j++ {
			res.Hist[i] += metrics[j].histValueFor(l, r)
		}
	}

	return res
}

// Returns percentile p meaning that p% values in histogram are less or equal to the return value.
// This function returns estimation based on the histogram, so don't expect to find the exact value.
func (qh *QuantHistogram) Percentile(p float32) float32 {
	if qh.Count == 0 {
		return 0.0
	}

	if int(qh.Count) == len(qh.Hist) {
		// We have values here
		svals := append([]float32(nil), qh.Hist...)
		sort.Slice(svals, func (i, j int) bool { return svals[i] < svals[j]; })

		idx := float32(qh.Count) * p / 100.0
		i1 := int(idx)
		if i1 >= len(qh.Hist) - 1 {
			return qh.Max
		}

		if i1 < 0 {
			return qh.Min
		}

		a := idx - float32(math.Trunc(float64(idx)))

		return svals[i1] * a + svals[i1 + 1] * (1.0 - a)
	}

	total := float32(0.0)
	for _, h := range qh.Hist {
		total += h
	}

	pb := total * p / 100.0
	total = 0.0
	delta := qh.Max - qh.Min
	if delta < math.SmallestNonzeroFloat32 {
		delta = 1.0
	}

	for i, h := range qh.Hist {
		if total + h >= pb {
			a := (pb - total) * float32(len(qh.Hist)) / delta
			if i + 1 <= len(qh.Hist) {
				return qh.Min + (float32(i) + a) * delta / float32(len(qh.Hist))
			}
		}
	}

	return qh.Max
}

// Serialize histogram value into binary data structure.
func (qh *QuantHistogram) Serialize() []byte {
	w := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(w)
	enc.Encode(qh)
	return w.Bytes()
}

// Deserialize histogram
func DecodeQuantHistogram(data []byte) (*QuantHistogram, error) {
	r := bytes.NewReader(data)
	dec := gob.NewDecoder(r)
	res := &QuantHistogram{}
	err := dec.Decode(res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Get index of the bin and ratio for value x
// This means that x = qh.Min + delta * i + delta * a
// If x is outside of the interval function returns -1, 0.0
func (qh *QuantHistogram) binForValue(x float32) (i int, a float32) {
	if x < qh.Min || x > qh.Max {
		return -1, 0.0
	}

	// Now lets calculate the bin:
	ix := float64(x - qh.Min) / float64(len(qh.Hist))
	i = int(math.Trunc(ix))
	a = float32(ix - math.Trunc(ix))
	for i >= len(qh.Hist) {
		i -= 1
		a += 1.0
	}
	return
}

func (qh *QuantHistogram) histValueFor(l, r float32) float32 {
	if l > qh.Max {
		return 0
	}
	if r < qh.Min {
		return 0
	}

	if int(qh.Count) == len(qh.Hist) {
		// We have values inside the histogram, not bins
		res := float32(0.0)
		for _, x := range qh.Hist {
			if x >= l && (x < r || (r == qh.Max && x == r)) {
				res += 1.0
			}
		}

		return res
	}

	il, al := qh.binForValue(l)
	ir, ar := qh.binForValue(r)

	if il < 0 {
		il = 0
		al = 0
	}

	if ir < 0 {
		ir = len(qh.Hist) - 1
		ar = 1.0
	}

	if il == ir {
		return float32(qh.Hist[il]) * (ar - al)
	} else {
		res := float32(qh.Hist[il]) * (1.0 - al)

		for i := 1; i < ir; i++ {
			res += qh.Hist[i]
		}

		res += float32(qh.Hist[ir]) * ar

		return res
	}
}

// Function implements Terrell–Scott rule to get number of bins with some corrections to be a little more precise
func binCountTerrellScott(cnt int) int {
	res := int(math.Pow(float64(cnt), 1.0 / 3.0)) + 1
	if res < 16 {
		res = 16
	}

	return res
}
