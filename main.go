package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/lomik/prometheus-png/pkg"
)

func main() {
	prom := flag.String("prometheus", "http://127.0.0.1:9090", "Prometheus addr")
	promPath := flag.String("prometheus.path", "/api/v1/query_range", "Path to query_range endpoint")
	listen := flag.String("listen", ":8080", "Listen addr")
	defaultTimeout := flag.Duration("timeout", 10*time.Second, "Default timeout for queries")

	flag.Parse()

	http.Handle("/", pkg.NewPNG(*prom, *promPath, *defaultTimeout))
	log.Fatal(http.ListenAndServe(*listen, nil))
}
