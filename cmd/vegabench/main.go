// cmd/vegabench/main.go
package main

import "flag"

func main() {

	numberOfOrders := flag.Int("orders", 50000, "Number of orders to benchmark")
	uniform := flag.Bool("uniform", false, "Use the same size for all orders")
	reportDuration := flag.String("reportDuration", "10s", "Report stats every so often (for syntax, see https://golang.org/pkg/time/#ParseDuration)")

	flag.Parse()

	BenchmarkMatching(*numberOfOrders, nil, false, *uniform, *reportDuration)
}
