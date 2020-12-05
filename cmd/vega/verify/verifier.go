package verify

import (
	"errors"
	"fmt"
	"regexp"
)

func verifier(params []string, f func(*reporter, []byte) string) error {
	if len(params) <= 0 {
		return errors.New("error: at least one file is required")
	}
	rprter := &reporter{}
	for i, v := range params {
		rprter.Start(v)
		bytes := readFile(rprter, v)
		if rprter.HasCurrError() {
			rprter.Dump("")
			continue
		}

		result := f(rprter, bytes)
		rprter.Dump(result)
		if i < len(params)-1 {
			fmt.Println()
		}
	}
	if rprter.HasError() {
		return errors.New("error: one or more file are ill formated or invalid")
	}
	return nil

}

func isValidParty(party string) bool {
	notIn := func(r rune) bool {
		var okchars = "abcdef0123456789"
		for _, v := range okchars {
			if v == r {
				return false
			}
		}
		return true
	}
	if len(party) != 64 {
		return false
	}

	for _, c := range party {
		if notIn(c) {
			return false
		}
	}

	return true
}

func isValidEthereumAddress(v string) bool {
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	return re.MatchString(v)
}
