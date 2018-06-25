package datastore

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestNotFoundReturnsTrueWhenIsNotFoundError(t *testing.T) {
	//a
	var err NotFoundError
	var isNotFound bool
	err = NotFoundError{}

	//a
	isNotFound = NotFound(err)

	//a
	assert.Equal(t, true, isNotFound)
}


func TestNotFoundReturnsFalseWhenIsOtherError(t *testing.T) {
	//a
	var err error
	var isNotFound bool

	//a
	isNotFound = NotFound(err)

	//a
	assert.Equal(t, false, isNotFound)
}
