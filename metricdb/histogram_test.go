package metricdb_test

import (
	"testing"
	"math/rand"
	"math"
	"runtime/debug"

	"github.com/rymis/nanostatsd/metricdb"
)

func cmpVal[T comparable](t *testing.T, a, b T) {
	if a != b {
		t.Errorf("CMP failed: %v != %v", a, b)
		debug.PrintStack()
	}
}

func TestHistogram(t *testing.T) {
	var x1 [13]float32 = [13]float32{
		0.1, 0.2, 0.3, 0.4, 0.5,
		2.1, 2.2, 2.3,
		4.1, 4.2, 4.3, 4.4, 4.5,
	}

	var x2 [15]float32 = [15]float32{
		0.5, 0.6, 0.7, 0.8, 0.9,
		2.5, 2.6, 2.7, 2.8, 2.9,
		4.5, 4.6, 4.7, 4.8, 4.9,
	}
	h1 := metricdb.NewQuantHistogram(x1[:])
	h2 := metricdb.NewQuantHistogram(x2[:])

	lx1, rx1, hh1 := h1.Histogram()
	cmpVal(t, len(hh1), 16)
	cmpVal(t, lx1, 0.1)
	cmpVal(t, rx1, 4.5)
	cmpVal(t, hh1[0], 3)
	cmpVal(t, hh1[1], 2)
	cmpVal(t, hh1[2], 0)

	lx2, rx2, hh2 := h2.Histogram()
	cmpVal(t, len(hh2), 16)
	cmpVal(t, lx2, 0.5)
	cmpVal(t, rx2, 4.9)
	cmpVal(t, hh2[0], 3)
	cmpVal(t, hh2[1], 2)
	cmpVal(t, hh2[2], 0)

	h := metricdb.MergeQuantHistograms([]metricdb.QuantHistogram{*h1, *h2})

	// Number of bins should be 4 now:
	cmpVal(t, len(h.Hist), 16)
	cmpVal(t, h.Hist[0], 3)
	cmpVal(t, h.Hist[1], 5)
	cmpVal(t, h.Hist[2], 2)
	cmpVal(t, h.Hist[3], 0)
	cmpVal(t, h.Min, 0.1)
	cmpVal(t, h.Max, 4.9)

	// And we can build the full histogram from all the values:
	ha := metricdb.NewQuantHistogram(append(x1[:], x2[:]...))
	cmpVal(t, len(ha.Hist), 16)
	cmpVal(t, ha.Hist[0], 3)
	cmpVal(t, ha.Hist[1], 5)
	cmpVal(t, ha.Hist[2], 2)
	cmpVal(t, ha.Hist[3], 0)
	cmpVal(t, ha.Min, 0.1)
	cmpVal(t, ha.Max, 4.9)
}

func TestMonteCarlo(t *testing.T) {
	for i := 0; i < 100; i++ {
		monteCarlo(t)
	}
}

func monteCarlo(t *testing.T) {
	// Generate bucket of random values:
	binCnt := rand.Intn(10) + 1
	valCnt := rand.Intn(10000) + 100
	centers := make([]float32, binCnt)
	values := make([]float32, valCnt)

	for i := 0; i < binCnt; i++ {
		centers[i] = rand.Float32() * 100.0 - 50.0
	}

	for i := 0; i < valCnt; i++ {
		bin := rand.Intn(binCnt)
		values[i] = float32(rand.NormFloat64() * 10) + centers[bin]
	}

	// OK, now we have the data. Let's make histograms:
	histograms := make([]metricdb.QuantHistogram, 0, 16)
	idx := 0
	l := 0
	for idx < valCnt {
		if idx + 16 >= valCnt {
			l = valCnt - idx
		} else {
			l = rand.Intn(40)
			if idx + l >= valCnt {
				l = valCnt - idx
			}
		}
		histograms = append(histograms, *metricdb.NewQuantHistogram(values[idx: idx + l]))
		idx += l
	}

	etalon := metricdb.NewQuantHistogram(values)
	hist := metricdb.MergeQuantHistograms(histograms)

	// Compare histograms
	cmpVal(t, etalon.Count, hist.Count)
	cmpVal(t, len(etalon.Hist), len(hist.Hist))
	cmpVal(t, etalon.Min, hist.Min)
	cmpVal(t, etalon.Max, hist.Max)

	if math.Abs(float64(etalon.Avg - hist.Avg)) > 0.0001 {
		t.Errorf("Invalid average value: %f != %f", hist.Avg, etalon.Avg)
	}
	if math.Abs(float64(etalon.Avg2 - hist.Avg2)) > 0.01 {
		t.Errorf("Invalid average^2 value: %f != %f", hist.Avg2, etalon.Avg2)
	}

	// Compare bins:
	delta := float64(0.0)
	for i := 0; i < len(hist.Hist); i++ {
		delta = math.Abs(float64(hist.Hist[i] - etalon.Hist[i]))
	}

	delta /= float64(len(hist.Hist))

	if delta > float64(valCnt) / 20 {
		t.Errorf("Too large difference between histograms: %f", delta)
	}
}
