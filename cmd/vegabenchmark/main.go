package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"code.vegaprotocol.io/vega/processor"
)

var (
	opts = struct {
		recordings string
		out        string
		times      int
	}{}

	//go:embed configs/*
	configsFS embed.FS
	markets   = []string{
		"configs/076BB86A5AA41E3E.json",
		"configs/1F0BB6EB5703B099.json",
		"configs/2839D9B2329C9E70.json",
		"configs/3C58ED2A4A6C5D7E.json",
		"configs/4899E01009F1A721.json",
		"configs/4899E01009F1A721.json",
		"configs/5A86B190C384997F.json",
	}
)

func init() {
	flag.StringVar(&opts.recordings, "recordings", "", "a coma separated list of paths to the vega recordings")
	flag.StringVar(&opts.out, "out", "bench.txt", "a file to store the outputs of the benchmarks")
	flag.IntVar(&opts.times, "times", 5, "how many time should the benchmarks run")
}

func main() {
	flag.Parse()
	if len(opts.recordings) <= 0 {
		reportError("missing recordings paths argument")
	}
	if opts.times <= 0 {
		reportError("times flag have to be > 0")
	}

	f, err := os.OpenFile(opts.out, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		reportError("could not open the output file")
	}

	recordings := strings.Split(opts.recordings, ",")
	for _, recording := range recordings {
		result := runBenchmark(recording)
		fmt.Fprint(f, result)
	}
}

func runBenchmark(recording string) string {
	var stats processor.Stats
	var totalOrders uint64
	benchResults := testing.Benchmark(func(b *testing.B) {
		b.Helper()
		b.N = opts.times //nolint:staticcheck
		for n := 0; n < b.N; n++ {
			// setup a new vega
			proc, bstats, err := setupVega()
			if err != nil {
				reportError(fmt.Sprintf("unable to initialize vega, %v\n", err))
			}
			stats = bstats
			// start replaying
			if err := replayAll(proc.Abci(), recording); err != nil {
				reportError(fmt.Sprintf("error replaying blockchain, %v\n", err))
			}
			totalOrders += stats.TotalOrders()
		}
	})

	return fmt.Sprintf("Benchmark%v\t%v\t%v\t%f orders/s\n",
		recording, benchResults, benchResults.MemString(), float64(totalOrders)/benchResults.T.Seconds())
}

func reportError(str string) {
	fmt.Printf("error: %v\n", str)
	flag.Usage()
	os.Exit(1)
}
