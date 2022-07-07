package domain

import "time"

type Request struct {
	HostID    string
	StartTime time.Time
	EndTime   time.Time
}
