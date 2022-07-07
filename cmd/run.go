package cmd

import (
	"context"
	"fmt"
	"github.com/lfordyce/tiger/internal/domain"
	"github.com/lfordyce/tiger/pkg/csv"
	"github.com/lfordyce/tiger/pkg/postgres"
	"github.com/lfordyce/tiger/pkg/queue"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sync"
	"time"
)

type ResultStats struct {
	worker     int
	elapsed    float64
	overhead   time.Duration
	hostnameID string
	startEnd   time.Time
	endTime    time.Time
}

type StreamWrite chan<- ResultStats

// cmdRun handles the `tiger run` sub-command
type cmdRun struct {
	gs *globalState
}

func (c *cmdRun) run(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("tiger needs at least one argument to load the test")
	}
	var err error
	config, err := getConfig(cmd.Flags())
	if err != nil {
		return err
	}

	pgconn, err := postgres.GetConfig(cmd.Flags())
	if err != nil {
		return fmt.Errorf("failed to parse postgres config cli flags: %w", err)
	}

	fmtProcess, err := domain.GetCsvConfig(cmd.Flags())
	if err != nil {
		return fmt.Errorf("failed to parse csv config cli flags: %w", err)
	}

	globalCtx, globalCancel := context.WithCancel(c.gs.ctx)
	defer globalCancel()

	c.gs.logger.WithField("workers", config.Workers).Info("concurrent worker count")
	file := stdinOrFile(args[0], c.gs.stdIn)

	processes := new(sync.WaitGroup)
	errCh := make(chan error, 1)
	resultCh := make(chan ResultStats, 10)

	repo, closeFunc, err := pgconn.OpenConnection(globalCtx)
	if err != nil {
		c.gs.logger.WithError(err).Error("Unable to connect to database")
		return err
	}
	qd := queue.NewDispatcher(config.Workers)
	go qd.Run()

	defer func() {
		closeFunc()
		qd.Stop()
	}()

	jq := &domain.QueueHandler{
		QueueJobHandler: domain.QueueJobHandlerFunc(func(job *domain.QueueJob) {
			// increment then WaitGroup when job is queued in dispatcher
			processes.Add(1)
			qd.Queue(job)
		}),
		TaskHandler: domain.TaskHandlerFunc(func(request domain.Request, u int) error {
			// decrements the WaitGroup when job is finished executing
			defer processes.Done()
			if _, err := LogDurationHandler(repo, u, c.gs.logger, resultCh).Process(request); err != nil {
				return err
			}
			return nil
		}),
	}

	elapsed := func() func() time.Duration {
		start := time.Now()
		return func() time.Duration {
			return time.Since(start)
		}
	}()

	fmtProcess.Run(csv.WithIoReader(file), jq, errCh)
	err = <-errCh
	if err != nil {
		return err
	}

	var local sync.WaitGroup
	var events []ResultStats
	local.Add(1)
	go func() {
		for event := range resultCh {
			events = append(events, event)
		}
		local.Done()
	}()
	processes.Wait()
	// signals that all events have been executed by the worker pool
	finished := elapsed()

	close(resultCh)
	local.Wait()

	c.gs.logger.WithField("total", len(events)).Info("total results collected")
	c.gs.logger.WithField("elapsed", finished).Info("execution time of all jobs")
	return nil
}

func (c *cmdRun) flagSet() *pflag.FlagSet {
	flags := pflag.NewFlagSet("", pflag.ContinueOnError)
	flags.SortFlags = false
	flags.IntP("workers", "w", 3, "Number of workers for concurrency work.")
	flags.String("user", "postgres", "Postgres user")
	flags.String("password", "password", "Postgres password")
	flags.String("host", "localhost", "Postgres hostname")
	flags.Uint16("port", 5432, "Postgres port")
	flags.String("database", "homework", "Postgres database name")
	flags.String("csv-host-hdr", "hostname", "The name of the CSV host id field")
	flags.String("csv-start-hdr", "start_time", "The name of the CSV start time field")
	flags.String("csv-end-hdr", "end_time", "The name of the CSV end time field")
	flags.String("csv-ts-fmt", "2006-01-02 15:04:05", "The go timestamp format of the CSV timestamp field")

	return flags
}

func getCmdRun(gs *globalState) *cobra.Command {
	c := &cmdRun{
		gs: gs,
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Start a benchmark runner",
		Long:  `Start a benchmark runner`,
		Args:  exactArgsWithMsg(1, "arg should either be \"-\", if reading data from stdin, or a path to a data file"),
		RunE:  c.run,
	}
	runCmd.Flags().SortFlags = false
	runCmd.Flags().AddFlagSet(c.flagSet())
	return runCmd
}
