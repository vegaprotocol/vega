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

package sqlstore

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// nolint:nakedret
func SanitizeSql(sql string, args ...any) (output string, err error) {
	replacer := func(match string) (replacement string) {
		n, _ := strconv.ParseInt(match[1:], 10, 0)
		switch arg := args[n-1].(type) {
		case string:
			return quoteString(arg)
		case int:
			return strconv.FormatInt(int64(arg), 10)
		case int8:
			return strconv.FormatInt(int64(arg), 10)
		case int16:
			return strconv.FormatInt(int64(arg), 10)
		case int32:
			return strconv.FormatInt(int64(arg), 10)
		case int64:
			return strconv.FormatInt(arg, 10)
		case time.Time:
			return quoteString(arg.Format("2006-01-02 15:04:05.999999 -0700"))
		case uint:
			return strconv.FormatUint(uint64(arg), 10)
		case uint8:
			return strconv.FormatUint(uint64(arg), 10)
		case uint16:
			return strconv.FormatUint(uint64(arg), 10)
		case uint32:
			return strconv.FormatUint(uint64(arg), 10)
		case uint64:
			return strconv.FormatUint(arg, 10)
		case float32:
			return strconv.FormatFloat(float64(arg), 'f', -1, 32)
		case float64:
			return strconv.FormatFloat(arg, 'f', -1, 64)
		case bool:
			return strconv.FormatBool(arg)
		case []byte:
			return `E'\\x` + hex.EncodeToString(arg) + `'`
		case []int16:
			var s string
			s, err = intSliceToArrayString(arg)
			return quoteString(s)
		case []int32:
			var s string
			s, err = intSliceToArrayString(arg)
			return quoteString(s)
		case []int64:
			var s string
			s, err = intSliceToArrayString(arg)
			return quoteString(s)
		case nil:
			return "null"
		default:
			err = fmt.Errorf("unable to sanitize type: %T", arg)
			return ""
		}
	}

	output = literalPattern.ReplaceAllStringFunc(sql, replacer)
	return
}

var literalPattern = regexp.MustCompile(`\$\d+`)

func quoteString(input string) (output string) {
	output = "'" + strings.Replace(input, "'", "''", -1) + "'"
	return
}

func intSliceToArrayString[T any](nums []T) (string, error) {
	w := strings.Builder{}
	w.WriteString("{")
	for i, n := range nums {
		if i > 0 {
			w.WriteString(",")
		}
		var intx int64
		switch n := any(n).(type) {
		case int16:
			intx = int64(n)
		case int32:
			intx = int64(n)
		case int64:
			intx = n
		default:
			return "", fmt.Errorf("unexpected type %T", n)
		}
		w.WriteString(strconv.FormatInt(intx, 10))
	}
	w.WriteString("}")
	return w.String(), nil
}
