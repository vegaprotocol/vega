package tests

import (
	"testing"
)

func BenchmarkMatching100(b *testing.B) {
	BenchmarkMatching(100, b, true, 1, true)
}

func BenchmarkMatching1000(b *testing.B) {
	BenchmarkMatching(1000, b, true, 1, true)
}

func BenchmarkMatching10000(b *testing.B) {
	BenchmarkMatching(10000, b, true, 1, true)
}

func BenchmarkMatching100000(b *testing.B) {
	BenchmarkMatching(100000, b, true, 1, true)
}


func BenchmarkMatching100Allocated(b *testing.B) {
	BenchmarkMatching(100, b, true, 0, true)
}

func BenchmarkMatching1000Allocated(b *testing.B) {
	BenchmarkMatching(1000, b, true, 0, true)
}

func BenchmarkMatching10000Allocated(b *testing.B) {
	BenchmarkMatching(10000, b, true, 0, true)
}

func BenchmarkMatching100000Allocated(b *testing.B) {
	BenchmarkMatching(100000, b, true, 0, true)
}

func BenchmarkMatching100Uniform(b *testing.B) {
	BenchmarkMatching(100, b, true, 1, false)
}

func BenchmarkMatching1000Uniform(b *testing.B) {
	BenchmarkMatching(1000, b, true, 1, false)
}

func BenchmarkMatching10000Uniform(b *testing.B) {
	BenchmarkMatching(10000, b, true, 1, false)
}

func BenchmarkMatching100000Uniform(b *testing.B) {
	BenchmarkMatching(100000, b, true, 1, false)
}