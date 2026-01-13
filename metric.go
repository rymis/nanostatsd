package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

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

	res.mux.HandleFunc("/values", func (resp http.ResponseWriter, req *http.Request) {
		q := req.URL.Query()
		nm := q.Get("name")
		if nm == "" {
			errResponse(resp, fmt.Errorf("Name is not specified"))
		}

		begin := parseQueryTime(q.Get("from"), time.Now().Add(-6 * time.Hour))
		end := parseQueryTime(q.Get("to"), time.Now())

		tagsArg := q.Get("tags")
		var tags []string
		if tagsArg != "" {
			tags = strings.Split(tagsArg, ",")
		}

		res := db.QueryHistogram(nm, tags, begin, end)

		jsonResponse(resp, res)
	})

	res.mux.HandleFunc("/counts", func (resp http.ResponseWriter, req *http.Request) {
		q := req.URL.Query()
		nm := q.Get("name")
		if nm == "" {
			errResponse(resp, fmt.Errorf("Name is not specified"))
		}

		begin := parseQueryTime(q.Get("from"), time.Now().Add(-6 * time.Hour))
		end := parseQueryTime(q.Get("to"), time.Now())

		tagsArg := q.Get("tags")
		var tags []string
		if tagsArg != "" {
			tags = strings.Split(tagsArg, ",")
		}

		res := db.QueryCounts(nm, tags, begin, end)

		jsonResponse(resp, res)
	})

	return res
}

// Write metric to the database
func (ss *SimpleStats) Add(msg *Message) {
	switch msg.Type {
	case MetricCounter:
		ss.DB.IncrementCount(msg.Name, msg.Value, msg.Tags...)
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

var floatTimeRx = regexp.MustCompile(`^[+-]?[0-9]?\.?[0-9]+([eE][+-]?[0-9]+)?$`)
var isoTimeRx = regexp.MustCompile(`^[0-9][0-9][0-9][0-9]-[01][0-9]-[0-3][0-9].*`)
func parseQueryTime(t string, def time.Time) time.Time {
	if floatTimeRx.MatchString(t) {
		ft, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return def
		}

		return time.UnixMilli(int64(ft))
	} else if isoTimeRx.MatchString(t) {
		d, err := time.Parse(time.RFC822, t)
		if err != nil {
			return def
		}

		return d
	}

	return def
}
