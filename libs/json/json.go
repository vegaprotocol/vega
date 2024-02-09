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

package json

import (
	"encoding/json"
	"fmt"
)

func Prettify(data interface{}) ([]byte, error) {
	bytes, err := json.MarshalIndent(data, "  ", "  ")
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func PrettifyStr(data interface{}) (string, error) {
	bytes, err := Prettify(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func PrettyPrint(data interface{}) error {
	prettifiedData, err := PrettifyStr(data)
	if err != nil {
		return err
	}
	fmt.Println(prettifiedData)
	return nil
}

func Print(data interface{}) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("unable to marshal message: %w", err)
	}

	fmt.Printf("%v\n", string(buf))

	return nil
}
