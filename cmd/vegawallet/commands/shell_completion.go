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
