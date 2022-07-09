package csv

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

type parsedData struct {
	hostname  string
	startTime time.Time
	endTime   time.Time
}

func TestWithCsvReader(t *testing.T) {
	csvInput := []byte(`
hostname,start_time,end_time
host_000008,2017-01-01 08:59:22,2017-01-01 09:59:22
host_000001,2017-01-02 13:02:02,2017-01-02 14:02:02
host_000008,2017-01-02 18:50:28,2017-01-02 19:50:28
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

	reader := WithIoReader(os.Stdin)
	var collect []parsedData
	for data := range reader.C() {

		start, err := time.Parse("2006-01-02 15:04:05", data.Get("start_time"))
		if err != nil {
			t.Fatal(err)
		}
		end, err := time.Parse("2006-01-02 15:04:05", data.Get("end_time"))
		if err != nil {
			t.Fatal(err)
		}

		p := parsedData{
			hostname:  data.Get("hostname"),
			startTime: start,
			endTime:   end,
		}
		collect = append(collect, p)
	}
	assert.Equal(t, len(collect), 3)
}

func TestCsvReaderCustomDelimiter(t *testing.T) {
	t.Parallel()
	csvInput := []byte(`
hostname;start_time;end_time
host_000008;2017-01-01 08:59:22;2017-01-01 09:59:22
host_000001;2017-01-02 13:02:02;2017-01-02 14:02:02
host_000008;2017-01-02 18:50:28;2017-01-02 19:50:28
`,
	)

	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pipeWriter.Write(csvInput); err != nil {
		t.Fatal(err)
	}
	if err := pipeWriter.Close(); err != nil {
		t.Fatal(err)
	}

	stdin := os.Stdin
	// Restore stdin right after the test.
	defer func() { os.Stdin = stdin }()
	os.Stdin = pipeReader

	reader := WithIoReaderAndDelimiter(os.Stdin, ';')
	var collect []parsedData
	for data := range reader.C() {
		start, err := time.Parse("2006-01-02 15:04:05", data.Get("start_time"))
		if err != nil {
			t.Fatal(err)
		}
		end, err := time.Parse("2006-01-02 15:04:05", data.Get("end_time"))
		if err != nil {
			t.Fatal(err)
		}

		p := parsedData{
			hostname:  data.Get("hostname"),
			startTime: start,
			endTime:   end,
		}
		collect = append(collect, p)
	}
	assert.Equal(t, len(collect), 3)
}
