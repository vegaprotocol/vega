package libs

import (
	"fmt"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"testing"
)

/*
BenchmarkMapIteration
values length: 8890
values length: 889000
values length: 88900000
values length: 1262157750
BenchmarkMapIteration-16    	  141975	      8371 ns/op
*/
func BenchmarkMapIteration(b *testing.B) {

	m := make(map[string]string)

	for c := 0; c < 1000; c++ {
		m[fmt.Sprintf("%d-key", c)] = fmt.Sprintf("%d-value", c)
	}

	lengthOfAllValues := 0

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		for _, v := range m {
			lengthOfAllValues += len(v)
		}
	}

	fmt.Printf("values length: %d\n", lengthOfAllValues)
}

/*
BenchmarkOrderedMapIteration
values length: 8890
values length: 889000
values length: 88900000
values length: 3754478140
BenchmarkOrderedMapIteration-16    	  422326	      2811 ns/op
*/
func BenchmarkOrderedMapIteration(b *testing.B) {

	m := orderedmap.New[string, string]()
	for c := 0; c < 1000; c++ {
		m.Set(fmt.Sprintf("%d-key", c), fmt.Sprintf("%d-value", c))
	}

	lengthOfAllValues := 0
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		for p := m.Oldest(); p != nil; p = p.Next() {
			lengthOfAllValues += len(p.Value)
		}
	}

	fmt.Printf("values length: %d\n", lengthOfAllValues)
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
BenchmarkMapDelete-16    	    6439	    182624 ns/op
*/
func BenchmarkMapDelete(b *testing.B) {

	m := make(map[string]string)

	var keys []string
	var values []string
	for c := 0; c < 100000; c++ {
		keys = append(keys, fmt.Sprintf("%d-key", c))
		values = append(values, fmt.Sprintf("%d-value", c))
	}

	for c := 0; c < 100000; c++ {
		m[keys[c]] = values[c]
	}

	b.ResetTimer()

	i := 0
	for n := 0; n < b.N; n++ {
		for c := 99999; c >= 0; c-- {
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
BenchmarkOrderedMapDelete-16    	    5668	    210437 ns/op
*/
func BenchmarkOrderedMapDelete(b *testing.B) {

	m := orderedmap.New[string, string]()

	var keys []string
	var values []string
	for c := 0; c < 100000; c++ {
		keys = append(keys, fmt.Sprintf("%d-key", c))
		values = append(values, fmt.Sprintf("%d-value", c))
	}

	for c := 0; c < 100000; c++ {
		m.Set(keys[c], values[c])
	}

	b.ResetTimer()

	i := 0
	for n := 0; n < b.N; n++ {
		for c := 99999; c >= 0; c-- {
			_, _ = m.Delete(keys[c])
			i++
		}
	}

	fmt.Printf("length: %d\n", m.Len())
}
