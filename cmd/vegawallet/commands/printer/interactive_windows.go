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

// enableLegacyWindowsANSI enables support for ANSI color sequences in the Windows
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
