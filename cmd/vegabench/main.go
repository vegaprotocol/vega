// cmd/vegabench/main.go
package main

import "flag"

func main() {

	numberOfOrders := flag.Int("orders", 50000, "Number of orders to benchmark")
	uniform := flag.Bool("uniform", false, "Use the same size for all orders")
	reportInterval := flag.Int("reportEvery", 0, "Report stats every n orders")

	flag.Parse()

	BenchmarkMatching(*numberOfOrders, nil, false, *uniform, *reportInterval)
}
