package cmd

import (
	"encoding/csv"
	"fmt"
	"github.com/lfordyce/tiger/internal/domain"
	"github.com/spf13/cobra"
	"io"
	"os"
	"sync"
)

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

func exactArgsWithMsg(n int, msg string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return fmt.Errorf("accepts %d arg(s), received %d: %s", n, len(args), msg)
		}
		return nil
	}
}

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
