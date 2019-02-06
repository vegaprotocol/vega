// cmd/vegabin/main.go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	cli := NewCli()
	cli.SetFlags()

	base := &command{cmd: cli.rootCmd, cli: cli}
	base.Cmd().SilenceErrors = true

	cli.AddCommand(base, &NodeCommand{})

	if err := cli.Run(); err != nil {
		// deal with ExitError, which should be recognize as error, and should
		// not be exit with status 0.
		if exitErr, ok := err.(ExitError); ok {
			if exitErr.Status != "" {
				fmt.Fprintln(os.Stderr, exitErr.Status)
			}
			if exitErr.Code == 0 {
				// when get error with ExitError, code should not be 0.
				exitErr.Code = 1
			}
			os.Exit(exitErr.Code)
		}

		// not ExitError, print error to os.Stderr, exit code 1.
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	go func() {
		sig := <-gracefulStop
		fmt.Printf("caught sig: %+v", sig)
		fmt.Println("Wait for 2 second to finish processing")
		time.Sleep(2*time.Second)
		os.Exit(0)
	}()
}

// ExitError defines exit error produce by cli commands.
type ExitError struct {
	Code   int
	Status string
}

// Error implements the error interface.
func (e ExitError) Error() string {
	return fmt.Sprintf("Exit Code: %d, Status: %s", e.Code, e.Status)
}