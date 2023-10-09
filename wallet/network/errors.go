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

package network

import "fmt"

type DoesNotExistError struct {
	Name string
}

func NewDoesNotExistError(n string) DoesNotExistError {
	return DoesNotExistError{
		Name: n,
	}
}

func (e DoesNotExistError) Error() string {
	return fmt.Sprintf("network \"%s\" doesn't exist", e.Name)
}
