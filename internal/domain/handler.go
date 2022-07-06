package domain

import (
	"context"
	"errors"
	"log"
)

type Handler interface {
	Process(Request) (float64, error)
}

type HandlerFunc func(Request) (float64, error)

func (hf HandlerFunc) Process(r Request) (float64, error) {
	return hf(r)
}

func ErrorChanHandler(h Handler, errc chan<- error) Handler {
	return HandlerFunc(func(r Request) (float64, error) {
		if _, err := h.Process(r); err != nil {
			errc <- err
		}
		return 0.0, nil
	})
}

// ShardHandler is used to process a request for a single shard.
type ShardHandler interface {
	Process(Request, int) error
}

type ShardHandlerFunc func(Request, int) error

func (shf ShardHandlerFunc) Process(r Request, n int) error {
	return shf(r, n)
}

type QueueJobHandler interface {
	Handle(*QueueJob)
}

type QueueJobHandlerFunc func(*QueueJob)

func (qjhf QueueJobHandlerFunc) Handle(qj *QueueJob) {
	qjhf(qj)
}

type QueueHandler struct {
	QueueJobHandler
	ShardHandler
}

func (qh *QueueHandler) Process(r Request, n int) error {
	qj := &QueueJob{
		r:  r,
		sh: qh.ShardHandler,
	}
	qh.QueueJobHandler.Handle(qj)
	return nil
}

type QueueJob struct {
	r       Request
	sh      ShardHandler
	retries uint64
}

func (qj *QueueJob) Execute(id int) error {
	if err := qj.sh.Process(qj.r, id); err != nil {
		return &QueueError{request: qj.r, attempt: qj.retries, worker: id, err: err}
	}
	return nil
}

func (qj *QueueJob) ShouldRetry(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		if qj.retries < 3 {
			qj.retries++
			return true
		}
	}
	return false
}

func (qj *QueueJob) Fail(err error) {
	log.Printf("failed to execute job %v\n", err)
}

type QueueError struct {
	request Request
	attempt uint64
	worker  int
	err     error
}

func (qe *QueueError) Error() string {
	return qe.err.Error()
}

func (qe *QueueError) Unwrap() error {
	return qe.err
}

func (qe *QueueError) Attempt() uint64 {
	return qe.attempt
}

func (qe *QueueError) Request() Request {
	return qe.request
}

func (qe *QueueError) Worker() int {
	return qe.worker
}
