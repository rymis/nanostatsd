package metricdb

import (
	"fmt"
	"sync"
)

type ShortTimeStorage struct {
	Lock sync.Mutex

	// Real time part:
	Values map[string][]float32
	Counts map[string]float32
	CurrentQuant Quant
	LastLTUpdate Quant

	// Middle term memory:
	MTS MetricStorage

	Queue chan cmdMsg
}

const (
	cmdMsgNop = uint(iota)
	cmdMsgMergeQuant
)

type cmdMsg struct {
	MsgType uint
	Quant Quant
	Values map[string][]float32
	Counts map[string]float32
}

func (ms *ShortTimeStorage) WriteValue(name string, value float32, tags ...string) {
	ms.Lock.Lock()
	defer ms.Lock.Unlock()

	ms.checkQuant()

	// Add direct metric:
	ms.Values[name] = append(ms.Values[name], value)

	// And up to 3 tag combinations
	for _, t1 := range tags {
		name1 := fmt.Sprintf("%s#%s", name, t1)
		ms.Values[name1] = append(ms.Values[name1], value)
		for _, t2 := range tags {
			if t1 < t2 {
				name2 := fmt.Sprintf("%s#%s", name1, t2)
				ms.Values[name2] = append(ms.Values[name2], value)

				for _, t3 := range tags {
					if t2 < t3 {
						name3 := fmt.Sprintf("%s#%s", name2, t3)
						ms.Values[name3] = append(ms.Values[name3], value)
					}
				}
			}
		}
	}
}

func (ms *ShortTimeStorage) IncrementCount(name string, value float32, tags ...string) {
	ms.Lock.Lock()
	defer ms.Lock.Unlock()

	ms.checkQuant()

	ms.Counts[name] += value

	// And up to 2 tag combinations
	for _, t1 := range tags {
		name1 := fmt.Sprintf("%s#%s", name, t1)
		ms.Counts[name1] += value
		for _, t2 := range tags {
			if t1 < t2 {
				name2 := fmt.Sprintf("%s#%s", name1, t2)
				ms.Counts[name2] += value

				for _, t3 := range tags {
					if t2 < t3 {
						name3 := fmt.Sprintf("%s#%s", name2, t2)
						ms.Counts[name3] += value
					}
				}
			}
		}
	}
}

func (ms *ShortTimeStorage) QueryHistogram(name string, begin, end Quant) []ExtendedQuantHistogram {
	ms.Lock.Lock()
	defer ms.Lock.Unlock()

	ms.checkQuant()

	ltEndQuant := end
	if ltEndQuant >= ms.CurrentQuant {
		ltEndQuant = ms.CurrentQuant - 1
	}

	var res []ExtendedQuantHistogram

	if ltEndQuant >= begin {
		res = ms.MTS.QueryHistogram(name, begin, ltEndQuant)
	}

	// Check if we need to add current metrics
	if end >= ms.CurrentQuant {
		values, ok := ms.Values[name]
		if ok {
			d := NewQuantHistogram(values)
			res = append(res, ExtendedQuantHistogram{
				Quant: ms.CurrentQuant,
				QuantHistogram: *d,
			})
		}
	}

	return res
}

func (ms *ShortTimeStorage) QueryCounts(name string, begin, end Quant) []QuantCount {
	ms.Lock.Lock()
	defer ms.Lock.Unlock()

	ms.checkQuant()

	ltEndQuant := end
	if ltEndQuant >= ms.CurrentQuant {
		ltEndQuant = ms.CurrentQuant - 1
	}

	var res []QuantCount

	if ltEndQuant >= begin {
		res = ms.MTS.QueryCounts(name, begin, ltEndQuant)
	}

	// Check if we need to add current metrics
	if end >= ms.CurrentQuant {
		cnt, ok := ms.Counts[name]
		if ok {
			res = append(res, QuantCount{
				Quant: ms.CurrentQuant,
				Count: cnt,
			})
		}
	}

	return res
}

func (ms *ShortTimeStorage) Close() error {
	close(ms.Queue)
	return ms.MTS.Close()
}

func (sts *ShortTimeStorage) ListMetrics(dst map[string]bool) {
	sts.Lock.Lock()
	defer sts.Lock.Unlock()

	for nm := range sts.Values {
		dst[nm] = true
	}

	for nm := range sts.Counts {
		dst[nm] = true
	}

	sts.MTS.ListMetrics(dst)
}

func (sts *ShortTimeStorage) Periodic() error {
	return nil
}

func newMetricStorage(mts MetricStorage) *ShortTimeStorage {
	res := &ShortTimeStorage{
		Values: make(map[string][]float32),
		Counts: make(map[string]float32),
		CurrentQuant: QuantNow(),
		MTS: mts,
		Queue: make(chan cmdMsg, 16),
	}

	go res.worker()

	return res
}

func (ms *ShortTimeStorage) checkQuant() {
	cur := QuantNow()
	if cur <= ms.CurrentQuant {
		// Time can change to lower value. We can't go back on timeline, so we just ignore this change.
		// The final price for this is worse values for 15 second interval, but we can pay this price.
		return
	}

	msg := cmdMsg{
		MsgType: cmdMsgMergeQuant,
		Quant: ms.CurrentQuant,
		Values: ms.Values,
		Counts: ms.Counts,
	}

	ms.Queue <- msg

	ms.Values = make(map[string][]float32)
	ms.Counts = make(map[string]float32)
	ms.CurrentQuant = cur
}

func (ms *ShortTimeStorage) worker() {
	for msg := range ms.Queue {
		switch msg.MsgType {
			case cmdMsgMergeQuant:
				for name, vals := range msg.Values {
					qh := NewQuantHistogram(vals)
					ms.MTS.WriteHistogram(name, msg.Quant, qh)
				}
				msg.Values = nil // Make it possible to free memory earlier

				for name, cnt := range msg.Counts {
					ms.MTS.WriteCounts(name, msg.Quant, cnt / 15.0)
				}
				msg.Counts = nil
			break;
		}
	}
}
