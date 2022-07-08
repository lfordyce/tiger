package cmd

import (
	"context"
	"fmt"
	"github.com/lfordyce/tiger/internal/domain"
	"github.com/lfordyce/tiger/pkg/consts"
	"github.com/lfordyce/tiger/pkg/log"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io"
	"io/ioutil"
	stdlog "log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"
)

// globalFlags contains global config values that apply sub-commands.
type globalFlags struct {
	noColor   bool
	logOutput string
	logFormat string
	verbose   bool
}

type globalState struct {
	ctx context.Context

	fs    afero.Fs
	getwd func() (string, error)
	args  []string

	defaultFlags, flags globalFlags

	outMutex       *sync.Mutex
	stdOut, stdErr *consoleWriter
	stdIn          io.ReadCloser

	osExit       func(int)
	signalNotify func(chan<- os.Signal, ...os.Signal)
	signalStop   func(chan<- os.Signal)

	logger         *logrus.Logger
	fallbackLogger logrus.FieldLogger
}

func newGlobalState(ctx context.Context) *globalState {
	isDumbTerm := os.Getenv("TERM") == "dumb"
	stdoutTTY := !isDumbTerm && (isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()))
	stderrTTY := !isDumbTerm && (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()))
	outMutex := &sync.Mutex{}
	stdOut := &consoleWriter{os.Stdout, colorable.NewColorable(os.Stdout), stdoutTTY, outMutex, nil}
	stdErr := &consoleWriter{os.Stderr, colorable.NewColorable(os.Stderr), stderrTTY, outMutex, nil}

	logger := &logrus.Logger{
		Out: stdOut,
		Formatter: &logrus.TextFormatter{
			ForceColors:   stdoutTTY,
			DisableColors: stdoutTTY,
		},
		Hooks: make(logrus.LevelHooks),
		Level: logrus.InfoLevel,
	}

	defaultFlags := getDefaultFlags()

	return &globalState{
		ctx:          ctx,
		fs:           afero.NewOsFs(),
		getwd:        os.Getwd,
		args:         append(make([]string, 0, len(os.Args)), os.Args...),
		defaultFlags: defaultFlags,
		flags:        defaultFlags,
		outMutex:     outMutex,
		stdOut:       stdOut,
		stdErr:       stdErr,
		stdIn:        os.Stdin,
		osExit:       os.Exit,
		signalNotify: signal.Notify,
		signalStop:   signal.Stop,
		logger:       logger,
		fallbackLogger: &logrus.Logger{
			Out:       stdErr,
			Formatter: new(logrus.TextFormatter),
			Hooks:     make(logrus.LevelHooks),
			Level:     logrus.InfoLevel,
		},
	}
}

func getDefaultFlags() globalFlags {
	return globalFlags{
		logOutput: "stderr",
	}
}

// This is to keep all fields needed for the main/root command
type rootCommand struct {
	globalState   *globalState
	cmd           *cobra.Command
	loggerStopped <-chan struct{}
}

func newRootCommand(gs *globalState) *rootCommand {
	c := &rootCommand{
		globalState: gs,
	}
	// the base command when called without any subcommands.
	rootCmd := &cobra.Command{
		Use:               "tiger",
		Short:             "benchmark sql query performance",
		SilenceUsage:      true,
		SilenceErrors:     true,
		PersistentPreRunE: c.persistentPreRunE,
	}

	rootCmd.PersistentFlags().AddFlagSet(rootCmdPersistentFlagSet(gs))
	rootCmd.SetArgs(gs.args[1:])
	rootCmd.SetOut(gs.stdOut)
	rootCmd.SetErr(gs.stdErr)
	rootCmd.SetIn(gs.stdIn)

	subCommands := []func(*globalState) *cobra.Command{
		getCmdRun, getCmdVersion,
	}

	for _, sc := range subCommands {
		rootCmd.AddCommand(sc(gs))
	}

	c.cmd = rootCmd
	return c
}

func (c *rootCommand) persistentPreRunE(*cobra.Command, []string) error {
	var err error

	c.loggerStopped, err = c.setupLoggers()
	if err != nil {
		return err
	}
	<-c.loggerStopped

	stdlog.SetOutput(c.globalState.logger.Writer())
	c.globalState.logger.Debugf("tiger version: v%s", consts.FullVersion())
	return nil
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

	c.globalState.logger.Error(err)
	exitCode := 1
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

	flags.BoolVarP(&gs.flags.verbose, "verbose", "v", gs.defaultFlags.verbose, "enable verbose logging")
	return flags
}

// RawFormatter it does nothing with the message just prints it
type RawFormatter struct{}

// Format renders a single log entry
func (f RawFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return append([]byte(entry.Message), '\n'), nil
}

// The returned channel will be closed when the logger has finished flushing and pushing logs after
// the provided context is closed. It is closed if the logger isn't buffering and sending messages
// Asynchronously
func (c *rootCommand) setupLoggers() (<-chan struct{}, error) {
	ch := make(chan struct{})
	close(ch)

	if c.globalState.flags.verbose {
		c.globalState.logger.SetLevel(logrus.DebugLevel)
	}

	loggerForceColors := false // disable color by default
	switch line := c.globalState.flags.logOutput; {
	case line == "stderr":
		loggerForceColors = !c.globalState.flags.noColor && c.globalState.stdErr.isTTY
		c.globalState.logger.SetOutput(c.globalState.stdErr)
	case line == "stdout":
		loggerForceColors = !c.globalState.flags.noColor && c.globalState.stdOut.isTTY
		c.globalState.logger.SetOutput(c.globalState.stdOut)
	case line == "none":
		c.globalState.logger.SetOutput(ioutil.Discard)

	case strings.HasPrefix(line, "file"):
		ch = make(chan struct{})
		hook, err := log.FileHookFromConfigLine(
			c.globalState.ctx, c.globalState.fs, c.globalState.getwd,
			c.globalState.fallbackLogger, line, ch,
		)
		if err != nil {
			return nil, err
		}

		c.globalState.logger.AddHook(hook)
		c.globalState.logger.SetOutput(ioutil.Discard)

	default:
		return nil, fmt.Errorf("unsupported log output '%s'", line)
	}

	switch c.globalState.flags.logFormat {
	case "raw":
		c.globalState.logger.SetFormatter(&RawFormatter{})
		c.globalState.logger.Debug("Logger format: RAW")
	case "json":
		c.globalState.logger.SetFormatter(&logrus.JSONFormatter{})
		c.globalState.logger.Debug("Logger format: JSON")
	default:
		c.globalState.logger.SetFormatter(&logrus.TextFormatter{
			ForceColors: loggerForceColors, DisableColors: c.globalState.flags.noColor,
		})
		c.globalState.logger.Debug("Logger format: TEXT")
	}
	return ch, nil
}

func LogDurationHandler(next domain.Handler, id int, logger *logrus.Logger, write StreamWrite) domain.Handler {
	return domain.HandlerFunc(func(r domain.Request) (result float64, err error) {
		defer func(start time.Time) {
			dur := time.Since(start)
			e := logger.WithFields(logrus.Fields{
				"worker_id":    id,
				"dur_ms":       dur,
				"query_dur_ms": result,
				"host_id":      r.HostID,
				"start_time":   r.StartTime,
				"end_time":     r.EndTime,
			})
			if err != nil {
				e.WithError(err).Error()
			} else {
				e.Debug("processing statistics")
				write <- domain.Sample{
					WorkerID:   id,
					Elapsed:    result,
					Overhead:   dur,
					HostnameID: r.HostID,
					StartTime:  r.StartTime,
					EndTime:    r.EndTime,
				}
			}
		}(time.Now())
		result, err = next.Process(r)
		return
	})
}
