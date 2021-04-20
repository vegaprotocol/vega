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
	genesisC = "config/genesis.json"

	selfPubKey = "074ddc82b509801bad2c4d40531e9353e5d7dc96465a5353883225c1dd60c49f"
)

func init() {
	flag.StringVar(&opts.recordings, "recordings", "", "a coma separated list of paths to the vega recordings")
}

func main() {
	flag.Parse()
	if len(opts.recordings) <= 0 {
		reportError("missing recordings paths argument")
	}

	recordings := strings.Split(opts.recordings, ",")
	for _, recording := range recordings {
		runBenchmark(recording)
	}
}

func runBenchmark(recording string) {
	var stats processor.Stats
	benchResults := testing.Benchmark(func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			// setup a new vega
			proc, bstats, err := setupVega(selfPubKey)
			if err != nil {
				reportError(fmt.Sprintf("unable to initialize vega, %v\n", err))
			}
			stats = bstats
			// start replaying
			if err := replayAll(proc.Abci(), recording); err != nil {
				reportError(fmt.Sprintf("error replaying blockchain, %v\n", err))
			}
		}
	})

	fmt.Printf("Benchmark%v\t%v\t%v\t%f orders/s\n",
		recording, benchResults, benchResults.MemString(), float64(stats.TotalOrders())/benchResults.T.Seconds())
}

func reportError(str string) {
	fmt.Printf("error: %v\n", str)
	flag.Usage()
	os.Exit(1)
}
