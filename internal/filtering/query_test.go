package filtering_test

import (
	"testing"

	"code.vegaprotocol.io/vega/internal/filtering"
	"github.com/stretchr/testify/assert"
)

func TestQueryFilter_ApplyEqualFilter(t *testing.T) {
	queryFilter := &filtering.QueryFilter{}

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

	assert.True(t, queryFilter.ApplyEqualFilter(x.A, y.B))
	assert.False(t, queryFilter.ApplyEqualFilter(x.A, z.C))
}

func TestQueryFilter_ApplyNotEqualFilter(t *testing.T) {
	queryFilter := &filtering.QueryFilter{}

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

	assert.True(t, queryFilter.ApplyNotEqualFilter(x.A, z.C))
	assert.False(t, queryFilter.ApplyNotEqualFilter(x.A, y.B))
}

func TestQueryFilter_ApplyRangeFilter(t *testing.T) {
	queryFilter := &filtering.QueryFilter{}
	queryFilterRange := &filtering.QueryFilterRange{Lower: uint64(10), Upper: uint64(40)}

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

	assert.True(t, queryFilter.ApplyRangeFilter(x.A, queryFilterRange, "uint64"))
	assert.False(t, queryFilter.ApplyRangeFilter(y.B, queryFilterRange, "int"))
}
