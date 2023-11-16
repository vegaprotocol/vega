// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package testing

import (
	"fmt"
	"os"
	"path/filepath"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/paths"

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

// ProtosEq is a gomock matcher for comparing messages for equality.
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

// AssertProtoEqual is a testing assertion that two protos are the same.
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
