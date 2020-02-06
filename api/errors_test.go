package api_test

import (
	"testing"

	"code.vegaprotocol.io/vega/api"
)

func TestErrorMapUniqueCodes(t *testing.T) {
	api.InitErrorMap()
	errors := api.ErrorMap

	existing := map[int32]bool{}
	for key, code := range errors {
		if _, ok := existing[code]; ok {
			t.Log("Duplicate code found in api.ErrorMap for code, duplicate =>", code, key)
			t.Fail()
			return
		}
		existing[code] = true
	}
}
