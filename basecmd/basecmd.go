package basecmd

import (
	"flag"
	"fmt"
	"os"
)

type Command struct {
	Name    string
	Long    string
	Short   string
	Run     func(args []string) int
	Usage   func()
	FlagSet *flag.FlagSet
}

func Main(cmds ...Command) {
	args := os.Args[1:]
	retval := 0

	if len(args) == 0 || args[0] == "help" {
		retval = printHelp(args, cmds)
	} else {
		var cmd Command
		var ok bool
		for _, v := range cmds {
			if v.Name == args[0] {
				cmd = v
				ok = true
				break
			}
		}
		if ok {
			retval = cmd.Run(args)
		} else {
			invalidCommand(args[0])
			retval = 1
		}

	}

	Exit(retval)
}

func invalidCommand(cmd string) {
	str := `vega %s: unknown command
Run 'vega help for usage.'
`
	fmt.Fprintf(os.Stderr, str, cmd)
}
