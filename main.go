package main

import (
	"fmt"
	"net/http"
	"flag"
)

// Very small and simple implementation of StatsD compatible statistics collector with very simple WebUI

func main() {
	httpAddr := flag.String("web", "localhost:8888", "Web interface address to use")
	statsdAddr := flag.String("listen", "localhost:8125", "Listend for statsd compatible stats on address")
	flag.Parse()

	stat := NewSimpleStats()
	http.Handle("/metrics", stat.Handler())
	http.Handle("/stats", stat)
	handleStaticPages()
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

	err := StatsDaemon(*statsdAddr, ch)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

