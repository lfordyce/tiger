package queue

import "testing"
import "errors"

func TestQuitNoJobs(t *testing.T) {
	q := NewDispatcher(1)
	go q.Run()

	q.Stop()
}

func TestErrorRetry(t *testing.T) {

	q := NewDispatcher(1)
	go q.Run()

	attempts := 0
	errRetry := errors.New("recoverable error")
	errFail := errors.New("non-recoverable error")

	quit := make(chan bool)

	q.Queue(NewJob(
		func(id int) error {
			attempts++
			if attempts == 1 {
				return errRetry
			}
			return errFail
		},
		func(err error) bool {
			return errors.Is(err, errRetry)
		},
		func(err error) {
			if errors.Is(err, errFail) {
				quit <- true
			}
		},
	))

	<-quit
	if attempts != 2 {
		t.Fatal("did not make 2 attempts")
	}
}

func TestNoJobExecuteFunc(t *testing.T) {
	q := NewDispatcher(1)
	go q.Run()

	quit := make(chan error)

	q.Queue(NewJob(
		nil,
		nil,
		func(err error) {
			quit <- err
		},
	))

	err := <-quit
	if err != ErrNoExecute {
		t.Fatalf("expected ErrNoExecute: %v", err)
	}

}

func TestNoJobRetryFunc(t *testing.T) {
	q := NewDispatcher(1)
	go q.Run()

	eee := errors.New("expected error")
	quit := make(chan error)

	q.Queue(NewJob(
		func(id int) error { return eee },
		nil,
		func(err error) {
			quit <- err
		},
	))

	err := <-quit
	if err != eee {
		t.Fatalf("expected 'expected error': %v", err)
	}
}
