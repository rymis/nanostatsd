package main

import (
	"fmt"
	"net/http"
	"flag"
	"mime"

	"github.com/rymis/nanostatsd/metricdb"
)

// Very small and simple implementation of StatsD compatible statistics collector with very simple WebUI

func main() {
	httpAddr := flag.String("web", "localhost:8888", "Web interface address to use")
	statsdAddr := flag.String("listen", "localhost:8125", "Listend for statsd compatible stats on address")
	static := flag.String("static", "", "Use this directory for serving static pages instead of statically compiled ones")
	store := flag.String("store", "metrics", "Use this directory to store metrics")
	flag.Parse()

	cfg := &metricdb.MetricsDBConfig{}
	cfg.Path = *store
	cfg.LongTermStorage = "sql"
	cfg.MiddleTermStorage = "sql"
	db, err := metricdb.NewMetricsDB(cfg)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	stat := NewSimpleStats(db)
	http.Handle("/api", http.StripPrefix("/api", stat))
	if *static == "" {
		handleStaticPages()
	} else {
		mime.AddExtensionType(".mjs", "text/javascript")
		http.Handle("/", http.FileServer(http.Dir(*static)))
	}

	go func () {
		http.ListenAndServe(*httpAddr, nil)
	}()

	ch := make(chan *Message, 1000)
	go func() {
		for msg := range(ch) {
			// Debug print:
			// fmt.Printf("Message: %#v\n", msg)
			stat.Add(msg)
		}
	}()

	err = StatsDaemon(*statsdAddr, ch)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

