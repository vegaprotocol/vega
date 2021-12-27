package monitoring

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
)

var (
	minVersion = semver.MustParse("0.35.0")
	maxVersion = semver.MustParse("0.36.0")
)

var defaultChainVersion = ChainVersion{
	Min: minVersion,
	Max: maxVersion,
}

// ChainVersion represents a required version for the chain.
type ChainVersion struct {
	Min semver.Version
	Max semver.Version
}

// Check validate that they chain respect the minimal and maximum versions required.
func (c ChainVersion) Check(vstr string) error {
	vstr = stripVPrefix(vstr)

	v, err := semver.Parse(vstr)
	if err != nil {
		return err
	}

	if v.LT(c.Min) {
		return fmt.Errorf("expected version greater than or equal to %v but got %v", c.Min, v)
	}

	if v.GTE(c.Max) {
		return fmt.Errorf("expected version less than %v but got %v", c.Max, v)
	}

	return nil
}

func stripVPrefix(vstr string) string {
	return strings.TrimPrefix(vstr, "v")
}
