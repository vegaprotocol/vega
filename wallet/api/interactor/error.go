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

package interactor

import "fmt"

func TraceIDMismatchError(expected, got string) error {
	return fmt.Errorf("the trace IDs between the request and the response mismatch: expected %q, got %q", expected, got)
}

func WrongResponseTypeError(expected, got InteractionName) error {
	return fmt.Errorf("the received response does not match the expected response type: expected %q, got %q", string(expected), string(got))
}

func InvalidResponsePayloadError(name InteractionName) error {
	return fmt.Errorf("the received response has not a valid %q payload", string(name))
}
