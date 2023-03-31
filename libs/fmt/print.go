// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package fmt

import (
	"fmt"
	"strings"
)

func PrettyPrint(data map[string]string) {
	for k, v := range data {
		fmt.Printf("%s:\n%s\n", k, v)
	}
}

func Escape(s string) string {
	escaped := strings.ReplaceAll(s, "\n", "")
	escaped = strings.ReplaceAll(escaped, "\r", "")
	return escaped
}
