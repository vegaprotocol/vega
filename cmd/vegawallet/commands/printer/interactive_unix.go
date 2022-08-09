//go:build !windows
// +build !windows

package printer

import (
	"io"

	"github.com/muesli/termenv"
)

func NewInteractivePrinter(w io.Writer) *InteractivePrinter {
	profile := termenv.EnvColorProfile()
	return &InteractivePrinter{
		writer:       w,
		profile:      profile,
		checkMark:    termenv.String("✓ ").Foreground(profile.Color("2")).String(),
		questionMark: termenv.String("? ").Foreground(profile.Color("5")).String(),
		crossMark:    termenv.String("✗ ").Foreground(profile.Color("1")).String(),
		bangMark:     termenv.String("! "),
		arrow:        termenv.String("➜ "),
	}
}
