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
