package rand_test

import (
	"math/rand"
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"

	"github.com/stretchr/testify/assert"
)

func TestRandomHelpers(t *testing.T) {
	t.Run("Create a random string succeeds", testCreatingNewRandomStringSucceeds)
	t.Run("Create a random bytes succeeds", testCreatingNewRandomBytesSucceeds)
}

func testCreatingNewRandomStringSucceeds(t *testing.T) {
	size := rand.Intn(100)
	randomStr := vgrand.RandomStr(size)
	assert.Len(t, randomStr, size)
}

func testCreatingNewRandomBytesSucceeds(t *testing.T) {
	size := rand.Intn(100)
	randomBytes := vgrand.RandomBytes(size)
	assert.Len(t, randomBytes, size)
}
