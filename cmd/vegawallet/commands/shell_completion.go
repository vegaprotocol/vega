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

package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

var completionLong = `To load completions:

Bash:  To load completions for each session, execute once:
-----  Linux:
       $ {{.Software}} shell completion bash > /etc/bash_completion.d/{{.Software}}
       MacOS:
       $ {{.Software}} shell completion bash > /usr/local/etc/bash_completion.d/{{.Software}}


Zsh:   If shell completion is not already enabled in your environment you will need
----   to enable it.  You can execute the following once:
       $ echo "autoload -U compinit; compinit" >> ~/.zshrc

       To load completions for each session, execute once:
       $ {{.Software}} shell completion zsh > "${fpath[1]}/_{{.Software}}"

       You will need to start a new shell for this setup to take effect.


Fish:  To load completions for each session, execute once:
-----  $ {{.Software}} shell completion fish > ~/.config/fish/completions/{{.Software}}.fish
`

func NewCmdShellCompletion(w io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:                   "completion [bash|zsh|fish|powershell]",
		Short:                 "Generate completion script",
		Long:                  completionLong,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				_ = cmd.Root().GenBashCompletion(w)
			case "zsh":
				_ = cmd.Root().GenZshCompletion(w)
			case "fish":
				_ = cmd.Root().GenFishCompletion(w, true)
			case "powershell":
				_ = cmd.Root().GenPowerShellCompletion(w)
			}
		},
	}
}
