// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package verify

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	purple = color.New(color.FgMagenta).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
)

type reporter struct {
	file         string
	hasError     bool
	hasCurrError bool
	reports      []string
}

func (r *reporter) HasError() bool {
	return r.hasError
}

func (r *reporter) HasCurrError() bool {
	return r.hasCurrError
}

func (r *reporter) Start(f string) {
	r.file = f
}

func (r *reporter) Dump(result string) {
	ok := green("OK")
	if r.hasCurrError {
		r.hasError = true
		ok = red("NOT OK")
	}
	fmt.Printf("%v: %v\n", r.file, ok)
	if len(result) > 0 {
		fmt.Printf("%v\n", result)
	}
	for _, v := range r.reports {
		fmt.Print(v)
	}

	r.reports = []string{}
	r.hasCurrError = false
	r.file = ""
}

func (r *reporter) Warn(s string, args ...interface{}) {
	r.reports = append(r.reports, fmt.Sprintf(fmt.Sprintf("%v%v\n", fmt.Sprintf("%v: ", purple("warn")), s), args...))
}

func (r *reporter) Err(s string, args ...interface{}) {
	r.hasCurrError = true
	r.reports = append(r.reports, fmt.Sprintf(fmt.Sprintf("%v%v\n", fmt.Sprintf("%v: ", red("error")), s), args...))
}
