package metricdb

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type MetricsDB struct {
	Lock sync.Mutex

	// Real time part:
	values map[string][]float32
	Counts map[string]float32
	CurrentQuant Quant
	LastLTUpdate Quant

	// Quarter minute storages. Maybe later I'll make it tunable
	qmMetrics DataStorage[QuantHistogram]
	qmCounts DataStorage[float64]

	// Quarter hour storages. Also can be tunable
	qhMetrics DataStorage[QuantHistogram]
	qhCounts DataStorage[float64]

	// Number of Quarter minute/hour intervals to store
	numOfQms int
	numOfQhs int

	queue chan cmdMsg

	tagsCache map[string]map[string]uint8
}

type MetricWithTags struct {
	Metric string `json:"metric"`
	Tags []string `json:"tags,omitempty"`
	HasCount bool `json:"has_count"`
	HasValue bool `json:"has_value"`
}

/// Quant + Count
type QuantCount struct {
	Quant Quant `json:"quant"`
	Count float32 `json:"count"`
}

type MetricsDBConfig struct {
	MiddleTermStorage string `json:"middle_term_storage"`
	LongTermStorage string `json:"long_term_storage"`
	Path string `json:"path"`
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

const hasCountFlag uint8 = 1
const hasValueFlag uint8 = 2

func (ms *MetricsDB) WriteValue(name string, value float32, tags ...string) {
	ms.Lock.Lock()
	defer ms.Lock.Unlock()

	ms.checkQuant()

	// Add direct metric:
	ms.values[name] = append(ms.values[name], value)

	// And up to 3 tag combinations
	for _, t1 := range tags {
		name1 := fmt.Sprintf("%s#%s", name, t1)
		ms.values[name1] = append(ms.values[name1], value)
		for _, t2 := range tags {
			if t1 < t2 {
				name2 := fmt.Sprintf("%s#%s", name1, t2)
				ms.values[name2] = append(ms.values[name2], value)

				for _, t3 := range tags {
					if t2 < t3 {
						name3 := fmt.Sprintf("%s#%s", name2, t3)
						ms.values[name3] = append(ms.values[name3], value)
					}
				}
			}
		}
	}
}

func (ms *MetricsDB) IncrementCount(name string, value float32, tags ...string) {
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

func (mdb *MetricsDB) QueryHistogram(name string, tags []string, begin, end time.Time) []ExtendedQuantHistogram {
	b := TimeToQuant(begin)
	e := TimeToQuant(end)
	// TODO: approximate value if we have more than 3 tags
	nm := make([]string, 0, 4)
	nm = append(nm, name)
	for i := 0; i < len(tags) && i < 3; i++ {
		nm = append(nm, tags[i])
	}

	if len(nm) > 2 {
		sort.Strings(nm[1:])
	}

	return mdb.queryHistogram(strings.Join(nm, "#"), b, e)
}

func (ms *MetricsDB) queryHistogram(name string, begin, end Quant) []ExtendedQuantHistogram {
	ms.Lock.Lock()
	defer ms.Lock.Unlock()

	ms.checkQuant()

	ltEndQuant := end
	if ltEndQuant >= ms.CurrentQuant {
		ltEndQuant = ms.CurrentQuant
	}

	var res []ExtendedQuantHistogram

	if ltEndQuant >= begin {
		res = queryStorage(ms.qmMetrics, name, begin, ltEndQuant)
	}

	if begin < ms.CurrentQuant - Quant(ms.numOfQms) {
		res = append(res, queryStorage(ms.qhMetrics, name, begin, ltEndQuant)...)
	}

	// Check if we need to add current metrics
	if end >= ms.CurrentQuant {
		values, ok := ms.values[name]
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

func queryStorage(s DataStorage[QuantHistogram], name string, begin, end Quant) []ExtendedQuantHistogram {
	data, err := s.Query(name, begin, end)
	if err != nil {
		log.Printf("Storage error: %v", err)
		return nil
	}

	res := make([]ExtendedQuantHistogram, len(data))

	for i := range data {
		res[i].Quant = data[i].Quant
		res[i].QuantHistogram = *data[i].Value
	}

	return res
}

func (mdb *MetricsDB) QueryCounts(name string, tags []string,  begin, end time.Time) []QuantCount {
	b := TimeToQuant(begin)
	e := TimeToQuant(end)
	// TODO: approximate value if we have more than 3 tags
	nm := make([]string, 0, 4)
	nm = append(nm, name)
	for i := 0; i < len(tags) && i < 3; i++ {
		nm = append(nm, tags[i])
	}

	if len(nm) > 2 {
		sort.Strings(nm[1:])
	}

	return mdb.queryCounts(strings.Join(nm, "#"), b, e)
}

func (ms *MetricsDB) queryCounts(name string, begin, end Quant) []QuantCount {
	ms.Lock.Lock()
	defer ms.Lock.Unlock()

	ms.checkQuant()

	ltEndQuant := end
	if ltEndQuant >= ms.CurrentQuant {
		ltEndQuant = ms.CurrentQuant - 1
	}

	var res []QuantCount

	if ltEndQuant >= begin {
		res = queryStorageCounts(ms.qmCounts, name, begin, ltEndQuant)
	}

	if begin < ms.CurrentQuant - Quant(ms.numOfQms) {
		res = append(res, queryStorageCounts(ms.qhCounts, name, begin, ltEndQuant)...)
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

func queryStorageCounts(s DataStorage[float64], name string, begin, end Quant) []QuantCount {
	data, err := s.Query(name, begin, end)
	if err != nil {
		log.Printf("Storage error: %v", err)
		return nil
	}

	res := make([]QuantCount, len(data))

	for i := range data {
		res[i].Quant = data[i].Quant
		res[i].Count = float32(*data[i].Value)
	}

	return res
}

func (ms *MetricsDB) Close() error {
	close(ms.queue)
	err1 := ms.qmMetrics.Close()
	err2 := ms.qmCounts.Close()
	err3 := ms.qhMetrics.Close()
	err4 := ms.qhCounts.Close()

	for _, err := range []error{err1, err2, err3, err4} {
		if err != nil {
			return err
		}
	}

	return nil
}

func (mdb *MetricsDB) ListMetrics() []MetricWithTags {
	mdb.Lock.Lock()
	defer mdb.Lock.Unlock()

	res := make([]MetricWithTags, 0, len(mdb.tagsCache))
	for m, tags := range mdb.tagsCache {
		mt := MetricWithTags{
			Metric: m,
		}

		for t, kind := range tags {
			mt.Tags = append(mt.Tags, t)
			if kind & hasCountFlag != 0 {
				mt.HasCount = true
			}
			if kind & hasValueFlag != 0 {
				mt.HasValue = true
			}
		}


		res = append(res, mt)
	}

	return res
}

func NewMetricsDB(cfg *MetricsDBConfig) (*MetricsDB, error) {
	if cfg == nil {
		cfg = &MetricsDBConfig{}
	}

	res := &MetricsDB{
		values: make(map[string][]float32),
		Counts: make(map[string]float32),
		CurrentQuant: QuantNow(),
		queue: make(chan cmdMsg, 16),
		numOfQms: 4 * 60 * 24, // 1 day of 15 second intervals
		numOfQhs: 4 * 24 * 180, // Half a year of metrics to store
		tagsCache: make(map[string]map[string]uint8),
	}

	path, err := filepath.Abs(".")
	if err != nil {
		return nil, err
	}
	if cfg.Path != "" {
		path, err = filepath.Abs(cfg.Path)
		if err != nil {
			return nil, err
		}
	}

	if cfg.LongTermStorage == "" || cfg.LongTermStorage == "sql" {
		res.qhMetrics, err = NewSqlStorage[QuantHistogram](filepath.Join(path, "qh_metrics"))
		if err != nil {
			return nil, err
		}

		res.qhCounts, err = NewSqlStorage[float64](filepath.Join(path, "qh_counts"))
		if err != nil {
			res.qhMetrics.Close()
			return nil, err
		}
	} else if cfg.LongTermStorage == "mem" {
		res.qhMetrics = NewMemStorage[QuantHistogram]()
		res.qhCounts = NewMemStorage[float64]()
	} else {
		return nil, fmt.Errorf("Unknown storage type: %s", cfg.LongTermStorage)
	}

	if cfg.MiddleTermStorage == "" || cfg.MiddleTermStorage == "sql" {
		res.qmMetrics, err = NewSqlStorage[QuantHistogram](filepath.Join(path, "qm_metrics"))
		if err != nil {
			res.qhMetrics.Close()
			res.qhCounts.Close()
			return nil, err
		}

		res.qmCounts, err = NewSqlStorage[float64](filepath.Join(path, "qm_counts"))
		if err != nil {
			res.qhMetrics.Close()
			res.qhCounts.Close()
			res.qmMetrics.Close()
			return nil, err
		}
	} else if cfg.MiddleTermStorage == "mem" {
		res.qmMetrics = NewMemStorage[QuantHistogram]()
		res.qmCounts = NewMemStorage[float64]()
	} else {
		return nil, fmt.Errorf("Unknown storage type: %s", cfg.MiddleTermStorage)
	}

	res.updateMetricCache()

	go res.worker()

	return res, nil
}

func (ms *MetricsDB) checkQuant() {
	cur := QuantNow()
	if cur <= ms.CurrentQuant {
		// Time can change to lower value. We can't go back on timeline, so we just ignore this change.
		// The final price for this is worse values for 15 second interval, but we can pay this price.
		return
	}

	msg := cmdMsg{
		MsgType: cmdMsgMergeQuant,
		Quant: ms.CurrentQuant,
		Values: ms.values,
		Counts: ms.Counts,
	}

	ms.queue <- msg

	ms.values = make(map[string][]float32)
	ms.Counts = make(map[string]float32)
	ms.CurrentQuant = cur
}

func (ms *MetricsDB) worker() {
	for msg := range ms.queue {
		switch msg.MsgType {
			case cmdMsgMergeQuant:
				for name, vals := range msg.Values {
					qh := NewQuantHistogram(vals)
					ms.qmMetrics.WriteValue(name, msg.Quant, qh)
				}
				msg.Values = nil // Make it possible to free memory earlier

				for name, cnt := range msg.Counts {
					val := float64(cnt / 15.0)
					ms.qmCounts.WriteValue(name, msg.Quant, &val)
				}
				msg.Counts = nil
			break;
		}
	}
}

func splitMetricTags(metric string) (string, []string) {
	ms := strings.Split(metric, "#")
	return ms[0], ms[1:]
}

func (mdb *MetricsDB) updateMetrics(metric string, tags []string, flag uint8) {
	m, ok := mdb.tagsCache[metric]
	if !ok {
		m = make(map[string]uint8)
		mdb.tagsCache[metric] = m
	}

	for _, t := range tags {
		m[t] |= flag
	}
}

func (mdb *MetricsDB) updateMetricCache() {
	var name string
	var tags []string

	for nm := range mdb.values {
		name, tags = splitMetricTags(nm)
		mdb.updateMetrics(name, tags, hasValueFlag)
	}

	for nm := range mdb.Counts {
		name, tags = splitMetricTags(nm)
		mdb.updateMetrics(name, tags, hasCountFlag)
	}

	ms, err := mdb.qmMetrics.ListMetrics()
	if err != nil {
		log.Printf("Error: can't get list of metrics: %v", err)
	} else {
		for _, m := range ms {
			name, tags = splitMetricTags(m)
			mdb.updateMetrics(name, tags, hasValueFlag)
		}
	}

	ms, err = mdb.qmCounts.ListMetrics()
	if err != nil {
		log.Printf("Error: can't get list of metrics: %v", err)
	} else {
		for _, m := range ms {
			name, tags = splitMetricTags(m)
			mdb.updateMetrics(name, tags, hasCountFlag)
		}
	}

	ms, err = mdb.qhMetrics.ListMetrics()
	if err != nil {
		log.Printf("Error: can't get list of metrics: %v", err)
	} else {
		for _, m := range ms {
			name, tags = splitMetricTags(m)
			mdb.updateMetrics(name, tags, hasValueFlag)
		}
	}

	ms, err = mdb.qhCounts.ListMetrics()
	if err != nil {
		log.Printf("Error: can't get list of metrics: %v", err)
	} else {
		for _, m := range ms {
			name, tags = splitMetricTags(m)
			mdb.updateMetrics(name, tags, hasCountFlag)
		}
	}
}
