package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
)

var (
	opts = struct {
		recordings string
	}{}

	//go:embed configs/*
	configs embed.FS
	markets = []string{
		"configs/076BB86A5AA41E3E.json",
		"configs/1F0BB6EB5703B099.json",
		"configs/2839D9B2329C9E70.json",
		"configs/3C58ED2A4A6C5D7E.json",
		"configs/4899E01009F1A721.json",
		"configs/4899E01009F1A721.json",
		"configs/5A86B190C384997F.json",
	}
	genesis = "config/genesis.json"
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
	benchResults := testing.Benchmark(func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			// setup a new vega
			proc, err := setupVega()
			if err != nil {
				reportError(fmt.Sprintf("unable to initialize vega, %v\n", err))
			}
			// start replaying
			replayAll(proc, recording)
		}
	})

	fmt.Printf("Benchmark%v\t%v\t%v\n", recording, benchResults, benchResults.MemString())
}

func reportError(str string) {
	fmt.Printf("error: %v\n", str)
	flag.Usage()
	os.Exit(1)
}
