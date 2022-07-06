package queue

import (
	"errors"
	"sync"
	"testing"
	"time"
)

var (
	errDurationTooShort = errors.New("total duration was less than expected")
	errDurationTooLong  = errors.New("total duration was greater than expected")
)

type delayJob struct {
	mu       *sync.Mutex
	counter  int
	sleepDur time.Duration
	post     func(runs int)
}

func (dj *delayJob) Execute(int) error {
	dj.mu.Lock()
	dj.counter++
	run := dj.counter
	dj.mu.Unlock()

	time.Sleep(dj.sleepDur)

	go dj.post(run)
	return nil
}

func (dj *delayJob) ShouldRetry(error) bool {
	return false
}

func (dj *delayJob) Fail(error) {
}

func newDelayJob(runs int, quit chan bool) *delayJob {
	dj := &delayJob{
		mu:       &sync.Mutex{},
		sleepDur: 100 * time.Millisecond,
		post: func(run int) {
			if run >= runs {
				quit <- true
			}
		},
	}
	return dj
}

func expectDuration(t0 time.Time, durMin, durMax time.Duration, t *testing.T) {
	d0 := time.Now().Sub(t0)

	if d0 < durMin {
		t.Error(errDurationTooShort)
	} else if d0 > durMax {
		t.Error(errDurationTooLong)
	}
}

func runWorkerJobs(workers int, jobs int) time.Time {
	quit := make(chan bool)

	q := NewDispatcher(workers)
	go q.Run()

	t0 := time.Now()

	dj := newDelayJob(jobs, quit)
	for job := 0; job < jobs; job++ {
		q.Queue(dj)
	}

	<-quit
	q.Stop()
	return t0
}

func Test1Worker1Job(t *testing.T) {
	t0 := runWorkerJobs(1, 1)
	expectDuration(t0, time.Millisecond, 120*time.Millisecond, t)
}

func Test1Worker2Job(t *testing.T) {
	t0 := runWorkerJobs(1, 2)
	expectDuration(t0, 2*time.Millisecond, 220*time.Millisecond, t)
}

func Test2Worker2Job(t *testing.T) {
	t0 := runWorkerJobs(2, 2)
	expectDuration(t0, 1*time.Millisecond, 120*time.Millisecond, t)
}

func Test100Worker100Job(t *testing.T) {
	t0 := runWorkerJobs(100, 100)
	expectDuration(t0, 1*time.Millisecond, 120*time.Millisecond, t)
}
func Test100dWorker500Job(t *testing.T) {
	t0 := runWorkerJobs(100, 500)
	expectDuration(t0, 500*time.Millisecond, 540*time.Millisecond, t)
}
