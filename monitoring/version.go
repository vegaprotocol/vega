package monitoring

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
)

var (
	minVersion = semver.MustParse("0.33.8")
	maxVersion = semver.MustParse("0.34.0")
)

var defaultChainVersion = ChainVersion{
	Min: minVersion,
	Max: maxVersion,
}

// ChainVersion represents a required version for the chain
type ChainVersion struct {
	Min semver.Version
	Max semver.Version
}

// Check validate that they chain respect the minimal version required
func (c ChainVersion) Check(vstr string) error {
	vstr = stripVPrefix(vstr)

	v, err := semver.Parse(vstr)
	if err != nil {
		return err
	}

	if v.LT(c.Min) {
		return fmt.Errorf("expected version greater than %v but got %v", c.Min, v)
	}

	if v.GE(c.Max) {
		return fmt.Errorf("expected version less than %v but got %v", c.Max, v)
	}

	return nil
}

func stripVPrefix(vstr string) string {
	if strings.HasPrefix(vstr, "v") {
		vstr = strings.TrimPrefix(vstr, "v")
	}
	return vstr
}
