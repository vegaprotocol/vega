package flags

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	vgos "code.vegaprotocol.io/vega/libs/os"

	"golang.org/x/crypto/ssh/terminal"
)

// Empty is used when a command or sub-command receives no argument and has no execution.
type Empty struct{}

var (
	ErrPassphraseDoNotMatch = errors.New("passphrase do not match")

	supportedOutputs = []string{
		"json",
		"human",
	}
)

type OutputFlag struct {
	Output Output `long:"output" description:"Specify the output format: json,human" default:"human" required:"true"`
}

func (f OutputFlag) GetOutput() (Output, error) {
	outputStr := string(f.Output)
	if !isSupportedOutput(outputStr) {
		return "", fmt.Errorf("unsupported output \"%s\"", outputStr)
	}
	if f.Output == "human" && vgos.HasNoTTY() {
		return "", errors.New("output \"human\" is not script-friendly, use \"json\" instead")
	}
	return f.Output, nil
}

func isSupportedOutput(output string) bool {
	for _, o := range supportedOutputs {
		if output == o {
			return true
		}
	}
	return false
}

type Output string

func (o Output) IsHuman() bool {
	return string(o) == "human"
}

func (o Output) IsJSON() bool {
	return string(o) == "json"
}

type VegaHomeFlag struct {
	VegaHome string `long:"home" description:"Path to the custom home for vega"`
}

type PassphraseFlag struct {
	PassphraseFile Passphrase `short:"p" long:"passphrase-file" description:"A file containing the passphrase for the wallet, if empty will prompt for input"`
}

type Passphrase string

func (p Passphrase) Get(prompt string, withConfirmation bool) (string, error) {
	if len(p) == 0 {
		if vgos.HasNoTTY() {
			return "", errors.New("passphrase-file flag required without TTY")
		}
		return p.getFromUser(prompt, withConfirmation)
	}

	return p.getFromFile(string(p))
}

func (p Passphrase) getFromUser(prompt string, withConfirmation bool) (string, error) {
	passphrase, err := promptForPassphrase(fmt.Sprintf("Enter %s passphrase:", prompt))
	if err != nil {
		return "", err
	}

	if withConfirmation {
		passphraseConfirmation, err := promptForPassphrase(fmt.Sprintf("Confirm %s passphrase:", prompt))
		if err != nil {
			return "", err
		}

		if passphrase != passphraseConfirmation {
			return "", ErrPassphraseDoNotMatch
		}
	}

	return passphrase, nil
}

func promptForPassphrase(msg string) (string, error) {
	fmt.Print(msg)
	password, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("failed to read passphrase input: %w", err)
	}
	fmt.Println()

	return string(password), nil
}

func (p Passphrase) getFromFile(path string) (string, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimRight(string(buf), "\n"), nil
}

type PromptString string

// Get returns a string if set or prompts user otherwise.
func (p PromptString) Get(prompt, name string) (string, error) {
	if len(p) == 0 {
		if vgos.HasNoTTY() {
			return "", fmt.Errorf("%s flag required without TTY", name)
		}
		return p.getFromUser(prompt)
	}

	return string(p), nil
}

func (p PromptString) getFromUser(prompt string) (string, error) {
	var s string
	fmt.Printf("Enter %s:", prompt)
	defer func() { fmt.Printf("\n") }()
	if _, err := fmt.Scanf("%s", &s); err != nil {
		return "", fmt.Errorf("failed read the input: %w", err)
	}

	return s, nil
}
