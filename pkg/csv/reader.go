package csv

import (
	"encoding/csv"
	"io"
)

// Reader provides a reader of CSV streams whose first record is a header describing each field.
type Reader interface {
	// Header Answers the header.
	Header() []string
	// C Answers a channel that iterates over a sequence of Records in the stream. The channel
	// remains open until an error is encountered or until the stream is exhausted.
	C() <-chan Record
	// Error Answers the error that caused the stream to close, if any.
	Error() error
	// Close the reader and release any resources associated with it.
	Close()
}

type reader struct {
	init   chan interface{}
	quit   chan interface{}
	header []string
	err    error
	io     <-chan Record
}

// WithIoReader creates a csv Reader from the specified io Reader.
func WithIoReader(io io.ReadCloser) Reader {
	csvReader := csv.NewReader(io)
	csvReader.FieldsPerRecord = -1
	return WithCsvReader(csvReader, io)
}

// WithIoReaderAndDelimiter creates a csv Reader from the specified io Reader.
func WithIoReaderAndDelimiter(io io.ReadCloser, delimiter rune) Reader {
	csvReader := csv.NewReader(io)
	csvReader.Comma = delimiter
	csvReader.FieldsPerRecord = -1
	return WithCsvReader(csvReader, io)
}

// WithCsvReader creates a csv reader from the specified encoding/csv Reader.
func WithCsvReader(r *csv.Reader, c io.Closer) Reader {
	ch := make(chan Record)
	result := &reader{
		init: make(chan interface{}),
		quit: make(chan interface{}),
		io:   ch,
	}
	go func() {
		defer close(ch)
		defer func() {
			if c != nil {
				e := c.Close()
				if result.err == nil {
					result.err = e
				}
			}
		}()
		// read header
		if h, e := r.Read(); e != nil {
			result.header = []string{}
			result.err = e
			close(result.init)
			return
		} else {
			result.header = h
			close(result.init)
		}
		builder := NewRecordBuilder(result.header)
		for {
			if a, e := r.Read(); e != nil {
				if e != io.EOF {
					result.err = e
				}
				break
			} else {
				select {
				case <-result.quit:
					break
				default:
				}
				ch <- builder(a)
			}
		}
	}()
	return result
}

func (reader *reader) Header() []string {
	<-reader.init
	return reader.header
}

func (reader *reader) Error() error {
	<-reader.init
	return reader.err
}

func (reader *reader) C() <-chan Record {
	return reader.io
}

func (reader *reader) Close() {
	close(reader.quit)
}
