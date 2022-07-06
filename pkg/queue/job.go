package queue

import (
	"errors"
)

var (
	ErrNoExecute     = errors.New("no execute function specified")
	_            Job = &funcJob{}
)

// Job interface defines the methods for running executing a job
type Job interface {
	Execute(id int) error
	ShouldRetry(err error) bool
	Fail(err error)
}

type funcJob struct {
	executeFunc     func(id int) error
	shouldRetryFunc func(err error) bool
	failFunc        func(err error)
}

func NewJob(
	executeFunc func(id int) error,
	shouldRetryFunc func(err error) bool,
	failFunc func(err error)) Job {

	return &funcJob{
		executeFunc:     executeFunc,
		shouldRetryFunc: shouldRetryFunc,
		failFunc:        failFunc,
	}
}

func (fj *funcJob) Execute(id int) error {
	if fj.executeFunc == nil {
		return ErrNoExecute
	}

	return fj.executeFunc(id)
}

func (fj *funcJob) ShouldRetry(err error) bool {
	if err == ErrNoExecute {
		return false
	}

	if fj.shouldRetryFunc != nil {
		return fj.shouldRetryFunc(err)
	}
	return false
}

func (fj *funcJob) Fail(err error) {
	if fj.failFunc != nil {
		fj.failFunc(err)
	}
}
