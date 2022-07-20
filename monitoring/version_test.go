// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package monitoring_test

import (
	"testing"

	"code.vegaprotocol.io/vega/monitoring"
	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"
)

var (
	minVersion = semver.MustParse("0.1.2")
	maxVersion = semver.MustParse("0.2.0")
)

func newTestChainVersion() monitoring.ChainVersion {
	return monitoring.ChainVersion{
		Min: minVersion,
		Max: maxVersion,
	}
}

func TestVersion(t *testing.T) {
	t.Run("version is ok", testVersionOK)
	t.Run("version is less than expected", testVersionLesser)
	t.Run("version is greater than expected", testVersionGreater)
	t.Run("version with invalid format", testVersionInvalidFormat)
}

func testVersionOK(t *testing.T) {
	c := newTestChainVersion()

	// v = min
	okVersion := "0.1.2"
	err := c.Check(okVersion)
	assert.Nil(t, err)

	// v with vprefix
	okVersion = "v0.1.2"
	err = c.Check(okVersion)
	assert.Nil(t, err)

	// v between min and max
	okVersion = "0.1.123"
	err = c.Check(okVersion)
	assert.Nil(t, err)
}

func testVersionLesser(t *testing.T) {
	c := newTestChainVersion()

	koVersion := "0.1.1"
	err := c.Check(koVersion)
	assert.NotNil(t, err)
	assert.Equal(t, "expected version greater than or equal to 0.1.2 but got 0.1.1", err.Error())
}

func testVersionGreater(t *testing.T) {
	c := newTestChainVersion()

	// v == max
	koVersion := "0.2.0"
	err := c.Check(koVersion)
	assert.NotNil(t, err)
	assert.Equal(t, "expected version less than 0.2.0 but got 0.2.0", err.Error())

	// v > max
	koVersion = "0.345.0"
	err = c.Check(koVersion)
	assert.NotNil(t, err)
	assert.Equal(t, "expected version less than 0.2.0 but got 0.345.0", err.Error())
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

func TestVersionWithSuffix(t *testing.T) {
	c := newTestChainVersion()

	koVersion := "0.1.3-sometext"
	err := c.Check(koVersion)
	assert.Nil(t, err)
}
