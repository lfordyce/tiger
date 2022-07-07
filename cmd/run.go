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
		return err
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
		ShardHandler: domain.ShardHandlerFunc(func(request domain.Request, u int) error {
			// decrements the WaitGroup when job is finished executing
			defer processes.Done()
			if _, err := LogDurationHandler(repo, u, c.gs.logger, resultCh).Process(request); err != nil {
				return err
			}
			return nil
		}),
	}

	q := &QueryFormatProcess{
		Hostname:  "hostname",
		StartTime: "start_time",
		EndTime:   "end_time",
		Format:    "2006-01-02 15:04:05",
	}

	elapsed := func() func() time.Duration {
		start := time.Now()
		return func() time.Duration {
			return time.Since(start)
		}
	}()

	q.Run(csv.WithIoReader(file), jq, errCh)
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
	close(resultCh)
	local.Wait()
	finished := elapsed()

	//fmt.Printf("%+v\n", events)
	c.gs.logger.WithField("total", len(events)).Info("total results collected")
	c.gs.logger.WithField("elapsed", finished).Info("execution time of all jobs")
	return nil
}

func (c *cmdRun) flagSet() *pflag.FlagSet {
	flags := pflag.NewFlagSet("", pflag.ContinueOnError)
	flags.SortFlags = false
	flags.IntP("workers", "w", 3, "number of workers for concurrency work")
	flags.String("user", "postgres", "postgres user")
	flags.String("password", "password", "postgres password")
	flags.String("host", "localhost", "postgres hostname")
	flags.Uint16("port", 5432, "postgres port")
	flags.String("database", "homework", "postgres database name")

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

type QueryFormatProcess struct {
	Hostname  string
	StartTime string
	EndTime   string
	Format    string // the format of the timestamp column (for format see documentation of go time.Parse())
}

func (q *QueryFormatProcess) Run(reader csv.Reader, handler domain.ShardHandler, errCh chan<- error) {
	errCh <- func() error {
		defer reader.Close()

		for data := range reader.C() {
			start, err := time.Parse(q.Format, data.Get(q.StartTime))
			if err != nil {
				return err
			}
			end, err := time.Parse(q.Format, data.Get(q.EndTime))
			if err != nil {
				return err
			}

			//wg.Add(1)
			r := domain.Request{
				HostID:    data.Get(q.Hostname),
				StartTime: start,
				EndTime:   end,
			}
			if err := handler.Process(r, 0); err != nil {
				return err
			}
		}
		return reader.Error()
	}()
}
