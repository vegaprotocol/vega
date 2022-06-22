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

package config

import (
	"fmt"
	"io/ioutil"
	"strings"

	"golang.org/x/crypto/ssh/terminal"
)

// Empty is used when a command or sub-command receives no argument and has no execution.
type Empty struct{}

type VegaHomeFlag struct {
	VegaHome string `long:"home" description:"Path to the custom home for vega"`
}

type PassphraseFlag struct {
	Passphrase Passphrase `short:"p" long:"passphrase" description:"A file containing the passphrase for the wallet, if empty will prompt for input"`
}

type Passphrase string

func (p Passphrase) Get(prompt string) (string, error) {
	if len(p) == 0 {
		return p.getFromUser(prompt)
	}

	return p.getFromFile(string(p))
}

func (p Passphrase) getFromUser(prompt string) (string, error) {
	fmt.Printf("Enter %s passphrase:", prompt)
	password, err := terminal.ReadPassword(0)
	if err != nil {
		return "", err
	}

	return string(password), nil
}

func (p Passphrase) getFromFile(path string) (string, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimRight(string(buf), "\n"), nil
}
