package domain

import (
	"github.com/lfordyce/tiger/pkg/csv"
	"os"
	"testing"
)

func TestQueryFormatProcess_Run(t *testing.T) {
	csvInput := []byte(`
hostname,start_time,end_time
host_000008,2017-0001-01 08:59:22,2017-01-01 09:159:a22
host_000001,2017-01-02 13:02:02,2017-01-02 14:02:02
host_000008,,2017-01-02 19:50:28
`,
	)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(csvInput); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	stdin := os.Stdin
	// Restore stdin right after the test.
	defer func() { os.Stdin = stdin }()
	os.Stdin = r

	q := &QueryFormatProcess{
		Hostname:  "hostname",
		StartTime: "start_time",
		EndTime:   "start_time",
		Format:    "2006-01-02 15:04:05",
	}
	errCh := make(chan error, 1)
	var collect []Request
	q.Run(csv.WithIoReader(os.Stdin), TaskHandlerFunc(func(request Request, i int) error {
		collect = append(collect, request)
		return nil
	}), errCh)
	err = <-errCh
	if err != nil {
		t.Logf("found and error: %v", err)
	}
	t.Logf("done: %v", collect)

}
