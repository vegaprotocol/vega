package paths

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/shared/paths"
)

var (
	ErrProvideAPathToExplain       = errors.New("please provide a path to explain")
	ErrProvideOnlyOnePathToExplain = errors.New("please provide only one path to explain")
)

type ExplainCmd struct{}

func (opts *ExplainCmd) Execute(args []string) error {
	if argsLen := len(args); argsLen == 0 {
		return ErrProvideAPathToExplain
	} else if argsLen > 1 {
		return ErrProvideOnlyOnePathToExplain
	}

	pathName := args[0]

	explanation, err := paths.Explain(pathName)
	if err != nil {
		return fmt.Errorf("couldn't explain path %s: %w", pathName, err)
	}

	fmt.Printf("%s\n", explanation)
	return nil
}
