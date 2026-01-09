package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rymis/nanostatsd/metricdb"
)

// Metric interoperation
type SimpleStats struct {
	DB *metricdb.MetricsDB
	mux *http.ServeMux
}

// Create new wrapper around SimpleStats
func NewSimpleStats(db *metricdb.MetricsDB) *SimpleStats {
	res := &SimpleStats{}
	res.DB = db
	res.mux = http.NewServeMux()
	res.mux.HandleFunc("/list_metrics", func (resp http.ResponseWriter, req *http.Request) {
		metrics := res.DB.ListMetrics()
		jsonResponse(resp, metrics)
	})

	return res
}

// Write metric to the database
func (ss *SimpleStats) Add(msg *Message) {
	switch msg.Type {
	case MetricCounter:
		ss.DB.WriteCount(msg.Name, msg.Value, msg.Tags...)
	default:
		ss.DB.WriteValue(msg.Name, msg.Value, msg.Tags...)
	}
}

func (ss *SimpleStats) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	ss.mux.ServeHTTP(resp, req)
}

func jsonRequest[T any](req *http.Request) (*T, error) {
	if req.Header.Get("content-type") != "application/json" {
		return nil, fmt.Errorf("Invalid content type")
	}

	decoder := json.NewDecoder(req.Body)
	res := new(T)

	err := decoder.Decode(res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func jsonResponse(resp http.ResponseWriter, val any) {
	resp.Header().Add("content-type", "application/json")
	resp.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(resp)
	encoder.Encode(&struct {
		Res any `json:"result"`
	}{val})
}

func errResponse(resp http.ResponseWriter, err error) {
	resp.Header().Add("content-type", "application/json")
	resp.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(resp)
	encoder.Encode(&struct {
		Err string `json:"error"`
	}{err.Error()})
}
