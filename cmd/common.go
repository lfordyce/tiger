package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
)

func exactArgsWithMsg(n int, msg string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return fmt.Errorf("accepts %d arg(s), received %d: %s", n, len(args), msg)
		}
		return nil
	}
}

func stdinOrFile(args string, stdin io.ReadCloser) io.ReadCloser {
	var err error
	r := stdin
	if args != "-" {
		r, err = os.Open(args)
		if err != nil {
			panic(err)
		}
	}
	return r
}

func printToStdout(gs *globalState, s string) {
	if _, err := fmt.Fprint(gs.stdOut, s); err != nil {
		gs.logger.Errorf("could not print '%s' to stdout: %s", s, err.Error())
	}
}
