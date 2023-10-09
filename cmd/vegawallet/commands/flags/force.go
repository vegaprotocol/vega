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

package flags

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func AreYouSure() bool {
	return YesOrNo("Are you sure?")
}

func DoYouApproveTx() bool {
	return YesOrNo("Do you approve this transaction?")
}

func YesOrNo(question string) bool {
	reader := bufio.NewReader(os.Stdin)
	defer fmt.Println()
	for {
		fmt.Print(question + " (y/n) ") //nolint:forbidigo

		answer, err := reader.ReadString('\n')
		if err != nil {
			panic(fmt.Errorf("couldn't read input: %w", err))
		}

		answer = strings.ToLower(strings.Trim(answer, " \r\n\t"))

		switch answer {
		case "yes", "y":
			return true
		case "no", "n":
			return false
		default:
			fmt.Printf("invalid answer %q, expect \"yes\" or \"no\"\n", answer) //nolint:forbidigo
		}
	}
}
