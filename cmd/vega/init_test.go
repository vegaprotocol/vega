package main_test

import (
	"testing"

	main "code.vegaprotocol.io/vega/cmd/vega"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	tempDir, tidy, err := storage.TempDir("TestInit")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %s", err.Error())
	}
	defer tidy()

	logger := logging.NewTestLogger()
	err = main.RunInit(tempDir, true, logger)
	assert.NoError(t, err)
}