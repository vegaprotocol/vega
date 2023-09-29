package libs

import (
	"fmt"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"testing"
)

/*
BenchmarkMapIteration
iterations: 1000
iterations: 100000
iterations: 10000000
iterations: 139970000
BenchmarkMapIteration-16    	  139970	      8536 ns/op
*/
func BenchmarkMapIteration(b *testing.B) {

	m := make(map[string]string)

	for c := 0; c < 1000; c++ {
		m[fmt.Sprintf("%d-key", c)] = fmt.Sprintf("%d-value", c)
	}
	b.ResetTimer()

	i := 0
	for n := 0; n < b.N; n++ {
		for _, _ = range m {
			i++
		}
	}

	fmt.Printf("iterations: %d\n", i)
}

/*
BenchmarkOrderedMapIteration
iterations: 1000
iterations: 100000
iterations: 10000000
iterations: 411097000
BenchmarkOrderedMapIteration-16    	  411097	      2811 ns/op
*/
func BenchmarkOrderedMapIteration(b *testing.B) {

	m := orderedmap.New[string, string]()
	for c := 0; c < 1000; c++ {
		m.Set(fmt.Sprintf("%d-key", c), fmt.Sprintf("%d-value", c))
	}
	b.ResetTimer()

	i := 0
	for n := 0; n < b.N; n++ {
		for p := m.Oldest(); p != nil; p = p.Next() {
			i++
		}
	}

	fmt.Printf("iterations: %d\n", i)
}

/*
BenchmarkMapAdd
iterations: 1000
iterations: 100000
iterations: 10000000
iterations: 82893000
BenchmarkMapAdd-16    	   82893	     14207 ns/op
*/
func BenchmarkMapAdd(b *testing.B) {

	m := make(map[string]string)

	var keys []string
	var values []string
	for c := 0; c < 1000; c++ {
		keys = append(keys, fmt.Sprintf("%d-key", c))
		values = append(values, fmt.Sprintf("%d-value", c))
	}
	b.ResetTimer()

	i := 0
	for n := 0; n < b.N; n++ {
		for c := 0; c < 1000; c++ {
			m[keys[c]] = values[c]
			i++
		}
	}

	fmt.Printf("iterations: %d\n", i)
}

/*
BenchmarkOrderedMapAdd
iterations: 1000
iterations: 100000
iterations: 10000000
iterations: 89872000
BenchmarkOrderedMapAdd-16    	   89872	     12770 ns/op
*/
func BenchmarkOrderedMapAdd(b *testing.B) {

	m := orderedmap.New[string, string]()

	var keys []string
	var values []string
	for c := 0; c < 1000; c++ {
		keys = append(keys, fmt.Sprintf("%d-key", c))
		values = append(values, fmt.Sprintf("%d-value", c))
	}
	b.ResetTimer()

	i := 0
	for n := 0; n < b.N; n++ {
		for c := 0; c < 1000; c++ {
			m.Set(keys[c], values[c])
			i++
		}
	}

	fmt.Printf("iterations: %d\n", i)
}

/*
BenchmarkMapGet
iterations: 1000
iterations: 100000
iterations: 10000000
iterations: 146986000
BenchmarkMapGet-16    	  146986	      7892 ns/op
*/
func BenchmarkMapGet(b *testing.B) {

	m := make(map[string]string)

	var keys []string
	var values []string
	for c := 0; c < 1000; c++ {
		keys = append(keys, fmt.Sprintf("%d-key", c))
		values = append(values, fmt.Sprintf("%d-value", c))
	}

	for c := 0; c < 1000; c++ {
		m[keys[c]] = values[c]
	}

	b.ResetTimer()

	i := 0
	for n := 0; n < b.N; n++ {
		for c := 0; c < 1000; c++ {
			_ = m[keys[c]]
			i++
		}
	}

	fmt.Printf("iterations: %d\n", i)
}

/*
BenchmarkOrderedMapGet
iterations: 1000
iterations: 100000
iterations: 10000000
iterations: 139746000
BenchmarkOrderedMapGet-16    	  139746	      8157 ns/op
*/
func BenchmarkOrderedMapGet(b *testing.B) {

	m := orderedmap.New[string, string]()

	var keys []string
	var values []string
	for c := 0; c < 1000; c++ {
		keys = append(keys, fmt.Sprintf("%d-key", c))
		values = append(values, fmt.Sprintf("%d-value", c))
	}

	for c := 0; c < 1000; c++ {
		m.Set(keys[c], values[c])
	}

	b.ResetTimer()

	i := 0
	for n := 0; n < b.N; n++ {
		for c := 0; c < 1000; c++ {
			_, _ = m.Get(keys[c])
			i++
		}
	}

	fmt.Printf("iterations: %d\n", i)
}

/*
BenchmarkMapDelete
length: 0
length: 0
length: 0
length: 0
BenchmarkMapDelete-16    	  651661	      1799 ns/op
*/
func BenchmarkMapDelete(b *testing.B) {

	m := make(map[string]string)

	var keys []string
	var values []string
	for c := 0; c < 1000; c++ {
		keys = append(keys, fmt.Sprintf("%d-key", c))
		values = append(values, fmt.Sprintf("%d-value", c))
	}

	for c := 0; c < 1000; c++ {
		m[keys[c]] = values[c]
	}

	b.ResetTimer()

	i := 0
	for n := 0; n < b.N; n++ {
		for c := 999; c >= 0; c-- {
			delete(m, keys[c])
			i++
		}
	}

	fmt.Printf("length: %d\n", len(m))
}

/*
BenchmarkOrderedMapDelete
length: 0
length: 0
length: 0
length: 0
BenchmarkOrderedMapDelete-16    	  577935	      2049 ns/op
*/
func BenchmarkOrderedMapDelete(b *testing.B) {

	m := orderedmap.New[string, string]()

	var keys []string
	var values []string
	for c := 0; c < 1000; c++ {
		keys = append(keys, fmt.Sprintf("%d-key", c))
		values = append(values, fmt.Sprintf("%d-value", c))
	}

	for c := 0; c < 1000; c++ {
		m.Set(keys[c], values[c])
	}

	b.ResetTimer()

	i := 0
	for n := 0; n < b.N; n++ {
		for c := 999; c >= 0; c-- {
			_, _ = m.Delete(keys[c])
			i++
		}
	}

	fmt.Printf("length: %d\n", m.Len())
}
