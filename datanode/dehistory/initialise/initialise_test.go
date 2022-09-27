package initialise

import (
	"testing"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/stretchr/testify/assert"
)

func TestSelectRootSegment(t *testing.T) {
	segments := map[string]*v2.HistorySegment{
		"1": {FromHeight: 1001, ToHeight: 2000},
		"2": {FromHeight: 1001, ToHeight: 3000},
		"3": {FromHeight: 1001, ToHeight: 4000},
		"4": {FromHeight: 1001, ToHeight: 4000},
		"5": {FromHeight: 1001, ToHeight: 3000},
		"6": {FromHeight: 1001, ToHeight: 2000},
	}

	rootSegment := SelectRootSegment(segments)
	assert.Equal(t, int64(4000), rootSegment.ToHeight)
}

func TestSelectRootSegmentWithOneSegment(t *testing.T) {
	segments := map[string]*v2.HistorySegment{
		"1": {FromHeight: 1001, ToHeight: 2000},
	}

	rootSegment := SelectRootSegment(segments)
	assert.Equal(t, int64(2000), rootSegment.ToHeight)
}
