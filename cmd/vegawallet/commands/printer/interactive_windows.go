package printer

import (
	"io"
	"os"
	"sync"

	"github.com/muesli/termenv"
	"golang.org/x/sys/windows"
)

var enableANSI sync.Once

func NewInteractivePrinter(w io.Writer) *InteractivePrinter {
	enableLegacyWindowsANSI()
	profile := termenv.EnvColorProfile()
	return &InteractivePrinter{
		writer:       w,
		profile:      profile,
		checkMark:    termenv.String("* ").Foreground(profile.Color("2")).String(),
		questionMark: termenv.String("? ").Foreground(profile.Color("5")).String(),
		crossMark:    termenv.String("x ").Foreground(profile.Color("1")).String(),
		bangMark:     termenv.String("! "),
		arrow:        termenv.String("> "),
	}
}

// enableANSIColors enables support for ANSI color sequences in the Windows
// default console (cmd.exe and the PowerShell application). Note that this
// only works with Windows 10. Also note that Windows Terminal supports colors
// by default.
//
// Thanks for LipGloss for the hack:
//
//	https://github.com/charmbracelet/lipgloss
func enableLegacyWindowsANSI() {
	enableANSI.Do(func() {
		stdout := windows.Handle(os.Stdout.Fd())
		var originalMode uint32

		windows.GetConsoleMode(stdout, &originalMode)
		windows.SetConsoleMode(stdout, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	})
}
