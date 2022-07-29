package test

import (
	"path/filepath"

	vgrand "code.vegaprotocol.io/shared/libs/rand"
)

func RandomPath() string {
	return filepath.Join("/tmp", "vega_tests", vgrand.RandomStr(10))
}
