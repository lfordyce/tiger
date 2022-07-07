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

// TaskHandler is used to process a request.
type TaskHandler interface {
	Process(Request, int) error
}

type TaskHandlerFunc func(Request, int) error

func (shf TaskHandlerFunc) Process(r Request, n int) error {
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
	TaskHandler
}

func (qh *QueueHandler) Process(r Request, _ int) error {
	qj := &QueueJob{
		r:  r,
		th: qh.TaskHandler,
	}
	qh.QueueJobHandler.Handle(qj)
	return nil
}

type QueueJob struct {
	r       Request
	th      TaskHandler
	retries uint64
}

func (qj *QueueJob) Execute(id int) error {
	if err := qj.th.Process(qj.r, id); err != nil {
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
