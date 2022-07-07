package csv

import (
	"fmt"
	"os"
	"testing"
	"time"
)

type parsedData struct {
	hostname  string
	startTime string
	endTime   string
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
	w.Close()

	stdin := os.Stdin
	// Restore stdin right after the test.
	defer func() { os.Stdin = stdin }()
	os.Stdin = r

	reader := WithIoReader(os.Stdin)
	fmt.Println(reader.Header())
	for data := range reader.C() {

		ts := data.Get("start_time")
		start, err := time.Parse("2006-01-02 15:04:05", ts)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("START: %+v\n", start)

		p := parsedData{
			hostname:  data.Get("hostname"),
			startTime: data.Get("start_time"),
			endTime:   data.Get("end_time"),
		}
		fmt.Printf("%+v\n", p)
	}
}
