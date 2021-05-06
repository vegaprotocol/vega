package commands_test

import "math/rand"

func RandomNegativeI64() int64 {
	return (rand.Int63n(1000) + 1) * -1
}

func RandomI64() int64 {
	return rand.Int63()
}

func RandomPositiveI64() int64 {
	return rand.Int63()
}

func RandomPositiveU64() uint64 {
	return rand.Uint64() + 1
}
