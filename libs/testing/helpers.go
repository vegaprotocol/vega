package testing

import (
	"fmt"
	"os"
	"path/filepath"

	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/shared/paths"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func NewVegaPaths() (paths.Paths, func()) {
	path := filepath.Join("/tmp", "vega-tests", vgrand.RandomStr(10))
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		panic(err)
	}
	return paths.New(path), func() { _ = os.RemoveAll(path) }
}

// ProtosEq is a gomock matcher for comparing messages for equality
func ProtosEq(message proto.Message) ProtoMatcher {
	return ProtoMatcher{message}
}

type ProtoMatcher struct {
	expected proto.Message
}

func (m ProtoMatcher) Matches(x interface{}) bool {
	msg, ok := x.(proto.Message)
	if !ok {
		return false
	}
	return proto.Equal(msg, m.expected)
}

func (m ProtoMatcher) String() string {
	return fmt.Sprintf("is equal to %v (%T)", m.expected, m.expected)
}

type tHelper interface {
	Helper()
}

// AssertProtoEqual is a testing assertion that two protos are the same
func AssertProtoEqual(t assert.TestingT, expected, actual proto.Message, msgAndArgs ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	if !proto.Equal(expected, actual) {
		return assert.Fail(t, fmt.Sprintf("Not equal: \n"+
			"expected: %v\n"+
			"actual  : %v", expected, actual), msgAndArgs...)
	}

	return true

}
