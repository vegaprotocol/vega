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

func isValidTMKey(key string) bool {
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
