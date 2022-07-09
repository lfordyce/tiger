package cmd

import (
	"context"
	"fmt"
	"github.com/lfordyce/tiger/internal/domain"
	"github.com/lfordyce/tiger/pkg/csv"
	"github.com/lfordyce/tiger/pkg/postgres"
	"github.com/lfordyce/tiger/pkg/queue"
	"github.com/lfordyce/tiger/pkg/statistics"
	"github.com/lfordyce/tiger/pkg/table"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io"
	"sync"
	"time"
)

const (
	hostnameHeader       = "HOSTNAME"
	totalCountNameHeader = "TOTAL_RUN"
	totalTimeNameHeader  = "TOTAL_TIME"
	minHeader            = "MIN"
	maxHeader            = "MAX"
	medianHeader         = "MEDIAN"
	averageHeader        = "AVG"
)

// StreamWrite provides write-only access to an domain.Sample object.
type StreamWrite chan<- statistics.Sample

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
	sampleCh := make(chan statistics.Sample, 10)

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
			if _, err := LogDurationHandler(repo, u, c.gs.logger, sampleCh).Process(request); err != nil {
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

	fmtProcess.Run(csv.WithIoReader(file), jq, c.gs.logger, errCh)
	err = <-errCh
	if err != nil {
		c.gs.logger.WithError(err).Error("csv processing failed")
		return err
	}

	var local sync.WaitGroup

	var samples []statistics.Sample

	local.Add(1)
	go func() {
		for sample := range sampleCh {
			samples = append(samples, sample)
		}
		local.Done()
	}()
	processes.Wait()
	// signals that all events have been executed by the worker pool
	finished := elapsed()

	close(sampleCh)
	local.Wait()

	c.gs.logger.WithField("total", len(samples)).Info("total results collected")
	c.gs.logger.WithField("elapsed", finished).Info("execution time of all jobs")

	collection := make(map[string]statistics.GroupedSample)
	for _, s := range samples {
		collection[s.HostnameID] = statistics.GroupedSample{
			HostnameID: s.HostnameID,
			Elapsed:    append(collection[s.HostnameID].Elapsed, s.Elapsed),
			Overhead:   append(collection[s.HostnameID].Overhead, s.Overhead),
		}
	}

	var dStats []dataStats
	for k, v := range collection {
		dStats = append(dStats, dataStats{
			hostName:  k,
			totalRun:  len(v.Elapsed),
			totalTime: statistics.Sum(v.Elapsed),
			minTime:   statistics.Min(v.Elapsed),
			maxTime:   statistics.Max(v.Elapsed),
			median:    statistics.Median(v.Elapsed),
			average:   statistics.Mean(v.Elapsed),
		})
	}
	//c.gs.logger.Info("•••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••")
	c.gs.logger.Info("BENCHMARK STATISTICS BY HOSTNAME")
	renderState(dStats, c.gs.stdOut)

	var final []float64
	for _, v := range samples {
		final = append(final, v.Elapsed)
	}

	finalOutput := buildFinalTable()
	finalOutput.Data = []table.Row{}
	finalOutput.Data = append(finalOutput.Data, []string{
		fmt.Sprint(len(samples)),
		fmt.Sprintf("%s", finished),
		fmt.Sprintf("%.4fms", statistics.Min(final)),
		fmt.Sprintf("%.4fms", statistics.Max(final)),
		fmt.Sprintf("%.4fms", statistics.Median(final)),
		fmt.Sprintf("%.4fms", statistics.Mean(final)),
	})
	//c.gs.logger.Info("•••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••")
	c.gs.logger.Info("TOTAL BENCHMARK STATISTICS")
	finalOutput.Render(c.gs.stdOut)
	//c.gs.logger.Info("•••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••")
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

type dataStats struct {
	hostName  string
	totalRun  int
	totalTime float64
	minTime   float64
	maxTime   float64
	median    float64
	average   float64
}

func renderState(statuses []dataStats, w io.Writer) {
	t := buildTable()
	t.Data = []table.Row{}
	for _, status := range statuses {
		status := status
		t.Data = append(t.Data, statsToTableRow(status))
	}
	t.Render(w)
}

func statsToTableRow(status dataStats) []string {
	return []string{
		status.hostName,
		fmt.Sprint(status.totalRun),
		fmt.Sprintf("%.4fms", status.totalTime),
		fmt.Sprintf("%.4fms", status.minTime),
		fmt.Sprintf("%.4fms", status.maxTime),
		fmt.Sprintf("%.4fms", status.median),
		fmt.Sprintf("%.4fms", status.average),
	}
}

func buildTable() table.Table {
	columns := []table.Column{
		{
			Header:    hostnameHeader,
			Width:     7,
			Flexible:  true,
			LeftAlign: true,
		},
		{
			Header: totalCountNameHeader,
			Width:  9,
		},
		{
			Header: totalTimeNameHeader,
			Width:  11,
		},
		{
			Header: minHeader,
			Width:  11,
		},
		{
			Header: maxHeader,
			Width:  11,
		},
		{
			Header: medianHeader,
			Width:  11,
		},
		{
			Header: averageHeader,
			Width:  11,
		},
	}
	t := table.NewTable(columns, []table.Row{})
	t.Sort = []int{0}
	return t
}

func buildFinalTable() table.Table {
	columns := []table.Column{
		{
			Header: totalCountNameHeader,
			Width:  9,
		},
		{
			Header: totalTimeNameHeader,
			Width:  11,
		},
		{
			Header: minHeader,
			Width:  11,
		},
		{
			Header: maxHeader,
			Width:  11,
		},
		{
			Header: medianHeader,
			Width:  11,
		},
		{
			Header: averageHeader,
			Width:  11,
		},
	}
	t := table.NewTable(columns, []table.Row{})
	t.Sort = []int{0}
	return t
}
