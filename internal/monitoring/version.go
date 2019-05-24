package monitoring

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
)

var (
	minVersion = semver.MustParse("0.31.5")
	maxVersion = semver.MustParse("0.32.0")
)

var defaultChainVersion = ChainVersion{
	Min: minVersion,
	Max: maxVersion,
}

type ChainVersion struct {
	Min semver.Version
	Max semver.Version
}

func (c ChainVersion) Check(vstr string) error {
	vstr = stripVPrefix(vstr)

	v, err := semver.Parse(vstr)
	if err != nil {
		return err
	}

	if v.LT(minVersion) {
		return fmt.Errorf("expected version greater than %v but got %v", minVersion, v)
	}

	if v.GE(maxVersion) {
		return fmt.Errorf("expexted version lesser than %v but got %v", maxVersion, v)
	}

	return nil
}

func stripVPrefix(vstr string) string {
	if strings.HasPrefix(vstr, "v") {
		vstr = strings.TrimPrefix(vstr, "v")
	}
	return vstr
}
