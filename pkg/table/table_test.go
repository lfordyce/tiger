package table

import (
	"fmt"
	"io"
	"os"
	"testing"
)

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
	t.Data = []Row{}
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

func buildTable() Table {
	columns := []Column{
		{
			Header:    "HOSTNAME",
			Width:     7,
			Flexible:  true,
			LeftAlign: true,
		},
		{
			Header: "TOTAL_RUN",
			Width:  9,
		},
		{
			Header: "TOTAL_TIME",
			Width:  11,
		},
		{
			Header: "MIN",
			Width:  11,
		},
		{
			Header: "MAX",
			Width:  11,
		},
		{
			Header: "MEDIAN",
			Width:  11,
		},
		{
			Header: "AVG",
			Width:  11,
		},
	}
	t := NewTable(columns, []Row{})
	t.Sort = []int{0}
	return t
}

func TestNewTable(t *testing.T) {
	data := []dataStats{
		{
			hostName:  "host_000005",
			totalRun:  10,
			totalTime: 23.34,
			minTime:   5.32,
			maxTime:   6.23,
			median:    4.23,
			average:   4.15,
		},
		{
			hostName:  "host_000001",
			totalRun:  34,
			totalTime: 23.34,
			minTime:   5.32,
			maxTime:   6.23,
			median:    4.23,
			average:   4.15,
		},
		{
			hostName:  "host_000002",
			totalRun:  3,
			totalTime: 23.34,
			minTime:   5.32,
			maxTime:   6.23,
			median:    4.23,
			average:   4.15,
		},
		{
			hostName:  "host_000003",
			totalRun:  8,
			totalTime: 23.34,
			minTime:   5.32,
			maxTime:   6.23,
			median:    4.23,
			average:   4.15,
		},
		{
			hostName:  "host_000004",
			totalRun:  6,
			totalTime: 23.34,
			minTime:   5.32,
			maxTime:   6.23,
			median:    4.23,
			average:   4.15,
		},
		{
			hostName:  "host_000008",
			totalRun:  22,
			totalTime: 23.34,
			minTime:   5.32,
			maxTime:   6.23,
			median:    4.23,
			average:   4.15,
		},
	}

	renderState(data, os.Stdout)
}
