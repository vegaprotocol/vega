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

package jsonrpc

import (
	"errors"
)

var (
	ErrOnlySupportJSONRPC2 = errors.New("the API only supports JSON-RPC 2.0")
	ErrMethodIsRequired    = errors.New("the method is required")
)

// Params is just a nicer way to describe what's passed to the handlers.
type Params interface{}

type Request struct {
	// Version specifies the version of the JSON-RPC protocol.
	// MUST be exactly "2.0".
	Version string `json:"jsonrpc"`

	// Method contains the name of the method to be invoked.
	Method string `json:"method"`

	// Params is a by-name Structured value that holds the parameter values to be
	// used during the invocation of the method. This member MAY be omitted.
	Params Params `json:"params,omitempty"`

	// ID is an identifier established by the Client that MUST contain a String.
	// If it is not included it is assumed to be a notification.
	// The Server MUST reply with the same value in the Response object if included.
	// This member is used to correlate the context between the two objects.
	ID string `json:"id,omitempty"`
}

func (r *Request) Check() error {
	if r.Version != VERSION2 {
		return ErrOnlySupportJSONRPC2
	}

	if r.Method == "" {
		return ErrMethodIsRequired
	}

	return nil
}

func (r *Request) IsNotification() bool {
	return r.ID == ""
}
