package config

import (
	"fmt"
	"io/ioutil"
	"strings"

	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"golang.org/x/crypto/ssh/terminal"
)

// Empty is used when a command or sub-command receives no argument and has no execution.
type Empty struct{}

type RootPathFlag struct {
	RootPath string `short:"r" long:"root-path" description:"Path of the root directory in which the configuration will be located" env:"VEGA_CONFIG"`
	VegaHome string `long:"home" description:"Path to the custom home for vega"`
}

func NewRootPathFlag() RootPathFlag {
	return RootPathFlag{
		RootPath: vgfs.DefaultVegaDir(),
	}
}

type PassphraseFlag struct {
	PassphraseFile Passphrase `short:"p" long:"passphrase-file" description:"A file containing the passphrase for the wallet, if empty will prompt for input"`
}

type Passphrase string

func (p Passphrase) Get(prompt string) (string, error) {
	if len(p) == 0 {
		return p.getFromUser(prompt)
	}

	return p.getFromFile(string(p))
}

func (p Passphrase) getFromUser(prompt string) (string, error) {
	fmt.Printf("please enter %s passphrase:", prompt)
	password, err := terminal.ReadPassword(0)
	fmt.Printf("\n")
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
