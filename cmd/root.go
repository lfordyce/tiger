package cmd

import (
	"github.com/lfordyce/tiger/internal/domain"
	"github.com/lfordyce/tiger/pkg/log"
	"time"
)

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
