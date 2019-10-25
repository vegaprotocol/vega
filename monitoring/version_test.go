package monitoring_test

import (
	"testing"

	"code.vegaprotocol.io/vega/monitoring"
	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"
)

var (
	minVersion = semver.MustParse("0.31.5")
	maxVersion = semver.MustParse("0.32.0")
)

func newTestChainVersion() monitoring.ChainVersion {
	return monitoring.ChainVersion{
		Min: minVersion,
		Max: maxVersion,
	}
}

func TestVersion(t *testing.T) {
	t.Run("version is ok", testVersionOK)
	t.Run("version is lesser than expected", testVersionLesser)
	t.Run("version is greater than expected", testVersionGreater)
	t.Run("version with invalid format", testVersionInvalidFormat)
}

func testVersionOK(t *testing.T) {
	c := newTestChainVersion()

	// v = min
	okVersion := "0.32.6"
	err := c.Check(okVersion)
	assert.Nil(t, err)

	// v with vprefix
	okVersion = "v0.32.6"
	err = c.Check(okVersion)
	assert.Nil(t, err)

	// v between min and max
	okVersion = "0.32.123"
	err = c.Check(okVersion)
	assert.Nil(t, err)
}

func testVersionLesser(t *testing.T) {
	c := newTestChainVersion()

	koVersion := "0.31.4"
	err := c.Check(koVersion)
	assert.NotNil(t, err)
	assert.Equal(t, "expected version greater than 0.32.6 but got 0.31.4", err.Error())
}

func testVersionGreater(t *testing.T) {
	c := newTestChainVersion()

	// v == max
	koVersion := "0.33.0"
	err := c.Check(koVersion)
	assert.NotNil(t, err)
	assert.Equal(t, "expexted version lesser than 0.33.0 but got 0.33.0", err.Error())

	// v > max
	koVersion = "0.367.0"
	err = c.Check(koVersion)
	assert.NotNil(t, err)
	assert.Equal(t, "expexted version lesser than 0.33.0 but got 0.367.0", err.Error())
}

func testVersionInvalidFormat(t *testing.T) {
	c := newTestChainVersion()

	// empty
	err := c.Check("")
	assert.NotNil(t, err)
	assert.Equal(t, "Version string empty", err.Error())

	// weird things
	err = c.Check("asdasdn%$$%^&")
	assert.NotNil(t, err)
	assert.Equal(t, "No Major.Minor.Patch elements found", err.Error())
}
