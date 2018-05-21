package vegabench

import "flag"

func main() {

	blockSize := flag.Int("block", 1, "Block size for timestamp increment")
	numberOfOrders := flag.Int("orders", 50000, "Number of orders to benchmark")
	uniform := flag.Bool("uniform", false, "Use the same size for all orders")
	reportInterval := flag.Int("reportEvery", 0, "Report stats every n orders")

	flag.Parse()

	BenchmarkMatching(*numberOfOrders, nil, false, *blockSize, *uniform, *reportInterval)
}