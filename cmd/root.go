package cmd

import (
	"context"
	"fmt"
	"github.com/lfordyce/tiger/internal/domain"
	"github.com/lfordyce/tiger/pkg/log"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"
)

// globalFlags contains global config values that apply sub-commands.
type globalFlags struct {
	quiet     bool
	noColor   bool
	logOutput string
	logFormat string
	verbose   bool
}

type globalState struct {
	ctx context.Context

	args []string

	defaultFlags, flags globalFlags

	outMutex       *sync.Mutex
	stdOut, stdErr *consoleWriter
	stdIn          io.Reader

	osExit       func(int)
	signalNotify func(chan<- os.Signal, ...os.Signal)
	signalStop   func(chan<- os.Signal)
}

func newGlobalState(ctx context.Context) *globalState {
	isDumbTerm := os.Getenv("TERM") == "dumb"
	stdoutTTY := !isDumbTerm && (isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()))
	stderrTTY := !isDumbTerm && (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()))
	outMutex := &sync.Mutex{}
	stdOut := &consoleWriter{os.Stdout, colorable.NewColorable(os.Stdout), stdoutTTY, outMutex, nil}
	stdErr := &consoleWriter{os.Stderr, colorable.NewColorable(os.Stderr), stderrTTY, outMutex, nil}

	defaultFlags := getDefaultFlags()

	return &globalState{
		ctx:          ctx,
		args:         append(make([]string, 0, len(os.Args)), os.Args...), // copy
		defaultFlags: defaultFlags,
		flags:        defaultFlags,
		outMutex:     outMutex,
		stdOut:       stdOut,
		stdErr:       stdErr,
		stdIn:        os.Stdin,
		osExit:       os.Exit,
		signalNotify: signal.Notify,
		signalStop:   signal.Stop,
	}
}

func getDefaultFlags() globalFlags {
	return globalFlags{
		logOutput: "stderr",
	}
}

// This is to keep all fields needed for the main/root command
type rootCommand struct {
	globalState *globalState

	cmd *cobra.Command
}

func newRootCommand(gs *globalState) *rootCommand {
	c := &rootCommand{
		globalState: gs,
	}
	// the base command when called without any subcommands.
	rootCmd := &cobra.Command{
		Use:           "tiger",
		Short:         "benchmark sql query performance",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().AddFlagSet(rootCmdPersistentFlagSet(gs))
	rootCmd.SetArgs(gs.args[1:])
	rootCmd.SetOut(gs.stdOut)
	rootCmd.SetErr(gs.stdErr)
	rootCmd.SetIn(gs.stdIn)

	subCommands := []func(*globalState) *cobra.Command{
		getCmdRun,
	}

	for _, sc := range subCommands {
		rootCmd.AddCommand(sc(gs))
	}

	c.cmd = rootCmd
	return c
}

func (c *rootCommand) execute() {
	ctx, cancel := context.WithCancel(c.globalState.ctx)
	defer cancel()
	c.globalState.ctx = ctx

	err := c.cmd.Execute()
	if err == nil {
		cancel()
		return
	}

	exitCode := 1
	fmt.Fprintf(os.Stderr, "failed to process cmd: %v\n", err)
	c.globalState.osExit(exitCode)
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	gs := newGlobalState(context.Background())
	newRootCommand(gs).execute()
}

func rootCmdPersistentFlagSet(gs *globalState) *pflag.FlagSet {
	flags := pflag.NewFlagSet("", pflag.ContinueOnError)
	flags.StringVar(&gs.flags.logOutput, "log-output", gs.flags.logOutput, "change the output for tiger logs, possible values are stderr,stdout,none,file[=./path.fileformat]")
	flags.Lookup("log-output").DefValue = gs.defaultFlags.logOutput

	flags.StringVar(&gs.flags.logFormat, "log-format", gs.flags.logFormat, "log output format")
	flags.Lookup("log-format").DefValue = gs.defaultFlags.logFormat

	flags.BoolVar(&gs.flags.noColor, "no-color", gs.flags.noColor, "disable colored output")
	flags.Lookup("no-color").DefValue = strconv.FormatBool(gs.defaultFlags.noColor)
	return flags
}

type Logger struct {
	log.Logger
}

func (l Logger) DurationHandler(next domain.Handler, id int) domain.Handler {
	return domain.HandlerFunc(func(r domain.Request) (result float64, err error) {
		defer func(start time.Time) {
			dur := time.Since(start)
			l.Info().
				Int("worker_id", id).
				Dur("dur_ms", dur).
				Float64("query_dur_ms", result).
				Str("host_id", r.HostID).
				Str("start_time", r.StartTime).
				Str("end_time", r.EndTime).
				Err(err).Msg("")
		}(time.Now())
		result, err = next.Process(r)
		return
	})
}
