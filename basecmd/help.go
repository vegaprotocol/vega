package basecmd

import (
	"fmt"
	"os"
	"strings"
)

func printHelp(args []string, cmds []Command) int {
	if len(args) <= 1 {
		fmt.Println(helpStr(cmds))
		return 0
	}

	var cmd Command
	var okcmd bool
	for _, v := range cmds {
		if v.Name == args[1] {
			cmd = v
			okcmd = true
			break
		}
	}

	if okcmd {
		cmd.Usage()
		return 0
	}

	fmt.Fprintf(os.Stderr, "vega help %s: unknown help topic. Run 'vega help'.\n", args[1])

	return -1
}

func helpStr(cmds []Command) string {
	helpStr := `
Vega is a core node implementation for the vega protocol

Usage: vega <command> [args]

Available Commands:

%s
Use "vega help <command>" for more information about a command.
`

	var cmdshelp string
	for _, v := range cmds {
		cmdshelp = fmt.Sprintf("%s  %-10s%s\n", cmdshelp, v.Name, v.Short)
	}

	return strings.TrimSpace(fmt.Sprintf(helpStr, cmdshelp))
}
