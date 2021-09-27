package testing

import (
	"os"
	"path/filepath"

	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/shared/paths"
)

func NewVegaPaths() (paths.Paths, func()) {
	path := filepath.Join("/tmp", "vega-tests", vgrand.RandomStr(10))
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		panic(err)
	}
	return paths.NewPaths(path), func() { _ = os.RemoveAll(path) }
}
