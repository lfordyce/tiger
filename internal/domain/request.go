package domain

import (
	//nolint
	//+gci:gocritic
	"fmt"
	"time"

	"github.com/lfordyce/tiger/pkg/csv"
	"github.com/spf13/pflag"
)

type Request struct {
	HostID    string
	StartTime time.Time
	EndTime   time.Time
}

func GetCsvConfig(flags *pflag.FlagSet) (*QueryFormatProcess, error) {
	hostHeader, err := flags.GetString("csv-host-hdr")
	if err != nil {
		return nil, fmt.Errorf("failed to parse csv-host-hdr flag: %w", err)
	}

	startHeader, err := flags.GetString("csv-start-hdr")
	if err != nil {
		return nil, fmt.Errorf("failed to parse csv-start-hdr flag: %w", err)
	}

	endHeader, err := flags.GetString("csv-end-hdr")
	if err != nil {
		return nil, fmt.Errorf("failed to parse csv-end-hdr flag: %w", err)
	}

	format, err := flags.GetString("csv-ts-fmt")
	if err != nil {
		return nil, fmt.Errorf("failed to parse csv-ts-fmt flag: %w", err)
	}

	return &QueryFormatProcess{
		Hostname:  hostHeader,
		StartTime: startHeader,
		EndTime:   endHeader,
		Format:    format,
	}, nil
}

type QueryFormatProcess struct {
	Hostname  string
	StartTime string
	EndTime   string
	Format    string // the format of the timestamp column (for format see documentation of go time.Parse())
}

func (q *QueryFormatProcess) Run(reader csv.Reader, handler TaskHandler, errCh chan<- error) {
	errCh <- func() error {
		defer reader.Close()

		for data := range reader.C() {
			start, err := time.Parse(q.Format, data.Get(q.StartTime))
			if err != nil {
				return fmt.Errorf("failed to parse start time: %w", err)
			}
			end, err := time.Parse(q.Format, data.Get(q.EndTime))
			if err != nil {
				return fmt.Errorf("failed to parse end time: %w", err)
			}

			r := Request{
				HostID:    data.Get(q.Hostname),
				StartTime: start,
				EndTime:   end,
			}
			if err := handler.Process(r, 0); err != nil {
				return fmt.Errorf("failed to process task handler request: %w", err)
			}
		}
		return reader.Error()
	}()
}
