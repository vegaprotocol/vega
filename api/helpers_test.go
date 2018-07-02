package api

import (
	"testing"
	"time"
	"regexp"
	"github.com/stretchr/testify/assert"
)

func TestUnixTimestamp(t *testing.T) {
	cases := []struct {
		expected uint64
		datetime string
	}{
		{1415792726371,"2014-11-12T11:45:26.371Z"},
		{1514768400000,"2018-01-01T01:00:00.000Z"},
		{406054800111,"1982-11-13T17:00:00.111Z"},
		{1591935315123,"2020-06-12T04:15:15.123Z"},
	}

	for _, c := range cases {
		layout := "2006-01-02T15:04:05.000Z"
		parsed, _ := time.Parse(layout , c.datetime)
		res := unixTimestamp(parsed)
		assert.Equal(t, res, c.expected)
	}
}

func TestNewGuid(t *testing.T) {
	guidAsString := newGuid()
	assert.NotEmpty(t, guidAsString)
	assert.True(t, isValidUUID(guidAsString))
}

func isValidUUID(uuid string) bool {
	r := regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
	return r.MatchString(uuid)
}
