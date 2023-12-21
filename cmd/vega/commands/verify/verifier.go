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
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

func verifier(params []string, f func(*reporter, []byte) string) error {
	if len(params) <= 0 {
		return errors.New("error: at least one file is required")
	}
	rprter := &reporter{}
	for i, v := range params {
		rprter.Start(v)
		bs := readFile(rprter, v)
		if rprter.HasCurrError() {
			rprter.Dump("")
			continue
		}

		result := f(rprter, bs)
		rprter.Dump(result)
		if i < len(params)-1 {
			fmt.Println()
		}
	}
	if rprter.HasError() {
		return errors.New("error: one or more files are malformed or invalid")
	}
	return nil
}

func unmarshal(r *reporter, bs []byte, i proto.Message) bool {
	u := jsonpb.Unmarshaler{
		AllowUnknownFields: false,
	}

	err := u.Unmarshal(bytes.NewBuffer(bs), i)
	if err != nil {
		r.Err("unable to unmarshal file: %v", err)
		return false
	}

	return true
}

func marshal(i proto.Message) string {
	m := jsonpb.Marshaler{
		Indent:       "  ",
		EmitDefaults: true,
	}
	buf, _ := m.MarshalToString(i)
	return buf
}

func readFile(r *reporter, path string) []byte {
	f, err := os.Open(path)
	if err != nil {
		r.Err("%v, no such file or directory", path)
		return nil
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		r.Err("unable to read file: %v", err)
		return nil
	}

	return bytes
}

func isValidParty(party string) bool {
	if len(party) != 64 {
		return false
	}

	_, err := hex.DecodeString(party)
	return err == nil
}

func isValidCometBFTKey(key string) bool {
	keybytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return false
	}
	return len(keybytes) == 32
}

func isValidEthereumAddress(v string) bool {
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	return re.MatchString(v)
}
