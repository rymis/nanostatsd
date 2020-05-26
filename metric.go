package main

import (
	"github.com/zserge/metric"
	"net/http"
	"encoding/json"
)

// Metric interoperation
type SimpleStats struct {
	Metrics map[string]metric.Metric
}

func NewSimpleStats() *SimpleStats {
	res := &SimpleStats{}
	res.Metrics = make(map[string]metric.Metric)

	return res
}

func (ss *SimpleStats) Add(msg *Message) {
	m, ok := ss.Metrics[msg.Name]
	if !ok {
		switch msg.Type {
		case MetricCounter:
			m = metric.NewCounter("15m2s", "1h15s", "1d1m")
		case MetricHistogram:
			m = metric.NewHistogram("15m2s", "1h15s", "1d1m")
		default:
			m = metric.NewGauge("15m2s", "1h15s", "1d1m")
		}

		ss.Metrics[msg.Name] = m
	}

	m.Add(float64(msg.Value))
}

func (ss *SimpleStats) Handler() http.Handler {
	return metric.Handler(func () map[string]metric.Metric {
		return ss.Metrics
	})
}

func (ss *SimpleStats) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	data, err := json.Marshal(ss.Metrics)
	if err != nil {
		resp.Header().Add("Content-Type", "text/plain")
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte("Error: " + err.Error()))
		return
	}

	resp.Header().Add("Content-Type", "application/json")
	resp.WriteHeader(http.StatusOK)
	resp.Write(data)
}

