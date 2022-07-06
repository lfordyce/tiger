package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lfordyce/tiger/internal/domain"
	"github.com/lfordyce/tiger/pkg/csv"
	"github.com/lfordyce/tiger/pkg/log"
	"github.com/lfordyce/tiger/pkg/queue"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
	"sync"
)

// psql -h localhost -U postgres -p 5432 -d homework
const defaultPostgresURL = "postgres://postgres:password@localhost:5432/homework?sslmode=disable"

type Repository struct {
	Conn *pgxpool.Pool
}

func (r Repository) Process(req domain.Request) (float64, error) {

	row := r.Conn.QueryRow(context.Background(), "SELECT * FROM bench($1::TEXT, $2::TIMESTAMPTZ, $3::TIMESTAMPTZ)",
		req.HostID, req.StartTime, req.EndTime)

	var elapsed float64

	err := row.Scan(&elapsed)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0.0, nil
	}
	if err != nil {
		return 0.0, fmt.Errorf("postgres.EventStore: failed to query events table: %w", err)
	}
	return elapsed, nil
}

// cmdRun handles the `tiger run` sub-command
type cmdRun struct {
	gs *globalState
}

func (c *cmdRun) run(cmd *cobra.Command, args []string) error {
	var err error
	fmt.Println("in run command...")
	if len(args) < 1 {
		return fmt.Errorf("tiger needs at least one argument to load the test")
	}

	globalWaitGroup := new(sync.WaitGroup)
	l := Logger{Logger: log.NewLogger()}
	l.Info().Str("TEST", "testing testing")

	conn, err := pgxpool.Connect(context.Background(), defaultPostgresURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		return err
	}
	qd := queue.NewDispatcher(10)
	go qd.Run()

	defer func() {
		conn.Close()
		qd.Stop()
	}()

	repo := Repository{Conn: conn}

	jq := &domain.QueueHandler{
		QueueJobHandler: domain.QueueJobHandlerFunc(func(job *domain.QueueJob) {
			qd.Queue(job)
		}),
		ShardHandler: domain.ShardHandlerFunc(func(request domain.Request, u int) error {
			defer globalWaitGroup.Done()
			if _, err := l.DurationHandler(repo, u).Process(request); err != nil {
				return err
			}
			return nil
		}),
	}

	csvfile, err := os.Open("/Users/lfordyce/Workspace/Go/tiger/query_params.csv")
	if err != nil {
		l.LogError("csv file open", err)
		return err
	}

	q := &QueryFormatProcess{
		Hostname:  "hostname",
		StartTime: "start_time",
		EndTime:   "end_time",
	}

	errCh := make(chan error, 1)
	q.Run(csv.WithIoReader(csvfile), jq, globalWaitGroup, errCh)
	err = <-errCh
	if err != nil {
		return err
	}

	fmt.Println("finished processing csv queries...")
	globalWaitGroup.Wait()
	fmt.Println("WAITING DONE...")
	return nil
}

func (c *cmdRun) flagSet() *pflag.FlagSet {
	flags := pflag.NewFlagSet("", pflag.ContinueOnError)
	flags.SortFlags = false
	flags.Int64P("workers", "w", 1, "number of workers for concurrency work")
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
