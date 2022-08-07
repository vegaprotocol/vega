package commands_test

import (
	"math/rand"
	"strconv"
)

func RandomNegativeI64() int64 {
	return (rand.Int63n(1000) + 1) * -1
}

func RandomNegativeI64AsString() string {
	return strconv.FormatInt(RandomNegativeI64(), 10)
}

func RandomI64() int64 {
	return rand.Int63()
}

func RandomPositiveI64() int64 {
	return rand.Int63()
}

func RandomPositiveI64Before(n int64) int64 {
	return rand.Int63n(n)
}

func RandomPositiveU32() uint32 {
	return rand.Uint32() + 1
}

func RandomPositiveU64() uint64 {
	return rand.Uint64() + 1
}

func RandomPositiveU64AsString() string {
	return strconv.FormatUint(RandomPositiveU64(), 10)
}

func RandomPositiveU64Before(n int64) uint64 {
	return uint64(rand.Int63n(n))
}

func StringPtr(s string) *string {
	return &s
}

func Int64Ptr(i int64) *int64 {
	return &i
}

func Uint64Ptr(u uint64) *uint64 {
	return &u
}
