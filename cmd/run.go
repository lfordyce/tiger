package cmd

import (
	"fmt"
	"github.com/lfordyce/tiger/internal/domain"
	"github.com/lfordyce/tiger/pkg/csv"
	"github.com/lfordyce/tiger/pkg/postgres"
	"github.com/lfordyce/tiger/pkg/queue"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sync"
)

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
	c.gs.logger.WithField("workers", config.Workers).Info("concurrent worker count")
	file := stdinOrFile(args[0], c.gs.stdIn)

	processes := new(sync.WaitGroup)

	repo, closeFunc, err := postgres.NewRepository()
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
			qd.Queue(job)
		}),
		ShardHandler: domain.ShardHandlerFunc(func(request domain.Request, u int) error {
			defer processes.Done()
			if _, err := LogDurationHandler(repo, u, c.gs.logger).Process(request); err != nil {
				return err
			}
			return nil
		}),
	}

	q := &QueryFormatProcess{
		Hostname:  "hostname",
		StartTime: "start_time",
		EndTime:   "end_time",
	}

	errCh := make(chan error, 1)
	q.Run(csv.WithIoReader(file), jq, processes, errCh)
	err = <-errCh
	if err != nil {
		return err
	}

	processes.Wait()
	return nil
}

func (c *cmdRun) flagSet() *pflag.FlagSet {
	flags := pflag.NewFlagSet("", pflag.ContinueOnError)
	flags.SortFlags = false
	flags.IntP("workers", "w", 1, "number of workers for concurrency work")
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
}

func (q *QueryFormatProcess) Run(reader csv.Reader, handler domain.ShardHandler, wg *sync.WaitGroup, errCh chan<- error) {
	errCh <- func() error {
		defer reader.Close()

		for data := range reader.C() {
			wg.Add(1)
			r := domain.Request{
				HostID:    data.Get(q.Hostname),
				StartTime: data.Get(q.StartTime),
				EndTime:   data.Get(q.EndTime),
			}
			if err := handler.Process(r, 0); err != nil {
				return err
			}
		}
		return reader.Error()
	}()
}
