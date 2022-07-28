package visor

import (
	"fmt"
	"strings"
)

// TODO make these functions more robust
type Args []string

func (a Args) Exists(name string) bool {
	for _, arg := range a {
		if strings.Contains(arg, name) {
			return true
		}
	}

	return false
}

func (a *Args) Set(name, value string) bool {
	if a.Exists(name) {
		return false
	}

	if name[0:2] != "--" {
		name = fmt.Sprintf("--%s", name)
	}

	*a = append(*a, name, value)

	return true
}
