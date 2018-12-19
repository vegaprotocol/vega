package vegatime

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"time"
	"fmt"
)

func TestStamp_NanoSeconds(t *testing.T) {
	var tests = []struct {
		input    uint64
		expected int64
	}{
		{0, 0},
		{12345, 0},
		{1257894000000000000, 0},
		{1542381868613963392, 613963392},
		{9542381868999935232, 999935232},
	}
	for _, tt := range tests {
		x := Stamp(tt.input)
		assert.Equal(t, tt.expected, x.NanoSeconds())
	}
}

func TestStamp_Seconds(t *testing.T) {
	var tests = []struct {
		input    uint64
		expected int64
	}{
		{0, 0},
		{12345, 0},
		{1257894000000000000, 1257894000},
		{1542381868613963392, 1542381868},
		{9542381868999935232, 9542381868},
	}
	for _, tt := range tests {
		x := Stamp(tt.input)
		assert.Equal(t, tt.expected, x.Seconds())
	}
}

func TestStamp_Rfc3339(t *testing.T) {
	var tests = []struct {
		input    uint64
		expected string
	}{
		{0, "1970-01-01T01:00:00+01:00"},
		{946688400000000000, "2000-01-01T01:00:00Z"},
		{1257894000000000000, "2009-11-10T23:00:00Z"},
		{1542381868613963392, "2018-11-16T15:24:28Z"},
		{9542381868999935232, "2272-05-21T05:37:48Z"},
	}
	for _, tt := range tests {
		x := Stamp(tt.input)
		assert.Equal(t, tt.expected, x.Rfc3339())
	}
}

func TestStamp_Rfc3339Nano(t *testing.T) {
	var tests = []struct {
		input    uint64
		expected string
	}{
		{0, "1970-01-01T01:00:00+01:00"},
		{946688400000000000, "2000-01-01T01:00:00Z"},
		{1257894000000000000, "2009-11-10T23:00:00Z"},
		{1542381868613963392, "2018-11-16T15:24:28.613963392Z"},
		{9542381868999935232, "2272-05-21T05:37:48.999935232Z"},
	}
	for _, tt := range tests {
		x := Stamp(tt.input)
		assert.Equal(t, tt.expected, x.Rfc3339Nano())
	}
}

func TestStamp_RoundToNearest(f *testing.T) {

	 o := uint64(1544050879298 * 1000000)
	 //i := uint64(1545158175835902621)
	 i := uint64(1544050879298000000)

	 fmt.Println(o)
	 


	 t := Stamp(i).Datetime()

	n := time.Now()

	j := uint64(t.UnixNano())
	v := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), (t.Minute()/15)*15, 0, 0, t.Location())
	x := uint64(v.UnixNano())

	d := 10/15 * 15
	fmt.Println(t)
	fmt.Println(j)
	fmt.Println(n.UnixNano())

	fmt.Println(v)
	fmt.Println(x)
	fmt.Println(d)

}