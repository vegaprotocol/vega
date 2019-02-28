package filtering

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryFilter_ApplyEqualFilter(t *testing.T) {
	queryFilter := &QueryFilter{}

	x := struct {
		A uint64
	}{
		12345,
	}

	y := struct {
		B uint64
	}{
		12345,
	}

	z := struct {
		C string
	}{
		"54321",
	}

	success := queryFilter.ApplyEqualFilter(x.A, y.B)
	failure := queryFilter.ApplyEqualFilter(x.A, z.C)

	assert.Equal(t, true, success)
	assert.Equal(t, false, failure)
}

func TestQueryFilter_ApplyNotEqualFilter(t *testing.T) {
	queryFilter := &QueryFilter{}

	x := struct {
		A uint64
	}{
		12345,
	}

	y := struct {
		B uint64
	}{
		12345,
	}

	z := struct {
		C string
	}{
		"54321",
	}

	success := queryFilter.ApplyNotEqualFilter(x.A, z.C)
	failure := queryFilter.ApplyNotEqualFilter(x.A, y.B)

	assert.Equal(t, true, success)
	assert.Equal(t, false, failure)
}

func TestQueryFilter_ApplyRangeFilter(t *testing.T) {
	queryFilter := &QueryFilter{}
	queryFilterRange := &QueryFilterRange{Lower: uint64(10), Upper: uint64(40)}

	x := struct {
		A uint64
	}{
		uint64(20),
	}

	y := struct {
		B uint64
	}{
		uint64(20),
	}

	success := queryFilter.ApplyRangeFilter(x.A, queryFilterRange, "uint64")
	failure := queryFilter.ApplyRangeFilter(y.B, queryFilterRange, "int")

	assert.Equal(t, true, success)
	assert.Equal(t, false, failure)
}

func TestQueryFilter_ApplyFilters(t *testing.T) {
	// todo(cdm)
}
