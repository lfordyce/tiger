package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
	"syscall"
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

// Trap Interrupts, SIGINTs and SIGTERMs and call the given.
func handleTestAbortSignals(gs *globalState, gracefulStopHandler, onHardStop func(os.Signal)) (stop func()) {
	sigC := make(chan os.Signal, 2)
	done := make(chan struct{})
	gs.signalNotify(sigC, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigC:
			gracefulStopHandler(sig)
		case <-done:
			return
		}

		select {
		case sig := <-sigC:
			if onHardStop != nil {
				onHardStop(sig)
			}
			gs.osExit(int(105))
		case <-done:
			return
		}
	}()

	return func() {
		close(done)
		gs.signalStop(sigC)
	}
}
