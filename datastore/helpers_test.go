package datastore

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestNotFoundReturnsTrueWhenIsNotFoundError(t *testing.T) {
	var err NotFoundError
	var isNotFound bool
	err = NotFoundError{}

	isNotFound = NotFound(err)

	assert.Equal(t, true, isNotFound)
}

func TestNotFoundReturnsFalseWhenIsOtherError(t *testing.T) {
	var err error
	var isNotFound bool

	isNotFound = NotFound(err)

	assert.Equal(t, false, isNotFound)
}
