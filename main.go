package main

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lfordyce/tiger/cmd"
	"github.com/lfordyce/tiger/internal/domain"
	"github.com/lfordyce/tiger/pkg/log"
	"github.com/lfordyce/tiger/pkg/queue"
	"io"
	"os"
	"sync"
)

// psql -h localhost -U postgres -p 5432 -d homework
const defaultPostgresURL = "postgres://postgres:password@localhost:5432/homework?sslmode=disable"

//func processCSV(rc io.Reader) (ch chan []string) {
//	ch = make(chan []string, 10)
//	go func() {
//		r := csv.NewReader(rc)
//		if _, err := r.Read(); err != nil { //read header
//			log.Fatal(err)
//		}
//		defer close(ch)
//		for {
//			rec, err := r.Read()
//			if err != nil {
//				if err == io.EOF {
//					break
//				}
//				log.Fatal(err)
//			}
//			ch <- rec
//		}
//	}()
//	return
//}

type Requestor interface {
	C() <-chan domain.Request
}

type RequestorFunc func() <-chan domain.Request

func (rf RequestorFunc) C() <-chan domain.Request {
	return rf()
}

func FileProcessor(rc io.Reader, wg *sync.WaitGroup) (Requestor, error) {
	ch := make(chan domain.Request, 32)
	go func() {
		r := csv.NewReader(rc)
		if _, err := r.Read(); err != nil { //read header
			fmt.Fprintf(os.Stderr, "failed to parse csv header: %v\n", err)
		}
		defer close(ch)
		for {
			rec, err := r.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				fmt.Fprintf(os.Stderr, "failed to parse csv data: %v\n", err)
			}
			wg.Add(1)
			ch <- domain.Request{
				HostID:    rec[0],
				StartTime: rec[1],
				EndTime:   rec[2],
			}
		}
	}()

	return RequestorFunc(func() <-chan domain.Request {
		return ch
	}), nil
}

func openStdinOrFile() io.Reader {
	var err error
	r := os.Stdin
	if len(os.Args) > 1 {
		r, err = os.Open(os.Args[1])
		if err != nil {
			panic(err)
		}
	}
	return r
}

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

func main() {
	globalWaitGroup := new(sync.WaitGroup)
	l := cmd.Logger{Logger: log.NewLogger()}
	ctx := context.Background()
	conn, err := pgxpool.Connect(ctx, defaultPostgresURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	repo := Repository{
		Conn: conn,
	}

	lookup := domain.ShardHandlerFunc(func(request domain.Request, u int) error {
		if _, err := l.DurationHandler(repo, u).Process(request); err != nil {
			return err
		}
		return nil
	})

	qd := queue.NewDispatcher("lookup", 10)
	go qd.Run()

	jq := &domain.QueueHandler{
		QueueJobHandler: domain.QueueJobHandlerFunc(func(job *domain.QueueJob) {
			qd.Queue(job)
		}),
		ShardHandler: domain.ShardHandlerFunc(func(request domain.Request, u int) error {
			defer func() {
				globalWaitGroup.Done()
			}()
			return lookup.Process(request, u)
		}),
	}

	csvfile, err := os.Open("query_params.csv")
	if err != nil {
		l.LogError("csv file open", err)
	}

	f, err := FileProcessor(csvfile, globalWaitGroup)
	if err != nil {
		if _, err := fmt.Fprintf(os.Stderr, "failed to process csv file: %v\n", err); err != nil {
			panic(err)
		}
		os.Exit(1)
	}

	//for {
	//	select {
	//	case proc, ok := <-f.C():
	//		if !ok {
	//			l.Info().Str("csv_processing", "finished")
	//			break
	//		}
	//		if err := jq.Process(proc, 0); err != nil {
	//			l.LogError("final reader", err)
	//		}
	//	case <-ctx.Done():
	//		panic(ctx.Err())
	//	}
	//}
	for proc := range f.C() {
		if err := jq.Process(proc, 0); err != nil {
			l.LogError("final reader", err)
		}
	}
	fmt.Println("finished processing csv queries...")
	globalWaitGroup.Wait()
	fmt.Println("WAITING DONE...")
	//<-ctx.Done()

}
