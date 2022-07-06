package queue

import (
	"time"
)

type worker struct {
	d        *dispatcher
	id       int
	jobQueue chan Job
	quit     chan bool
}

func newWorker(d *dispatcher, id int) *worker {
	return &worker{
		d:        d,
		id:       id,
		jobQueue: make(chan Job),
		quit:     make(chan bool, 1),
	}
}

// run starts listening to the job queue for new jobs, blocking until the quit method is called. The dispatcher
// should queue a job in a new goroutine.
func (w *worker) run() {
	for {
		// Register this worker's job channel to the dispatcher's
		// worker pool. The dispatcher is expected to use a
		// buffered channel, so this will return immediately.
		w.d.workerPool <- w.jobQueue

		// When the dispatcher selects this worker's job channel
		// from the pool, execute the job. If the quit channel is
		// signaled, the method will return before a job is queued.
		select {
		case j := <-w.jobQueue:
			// work request received, execute the job and handle failures appropriately
			if err := j.Execute(w.id); err != nil {
				if j.ShouldRetry(err) {
					go func() {
						// TODO: make this a configurable parameter
						time.Sleep(200 * time.Millisecond)
						w.d.Queue(j)
					}()
				} else {
					j.Fail(err)
				}
			}
		case <-w.quit:
			close(w.jobQueue)
			close(w.quit)
			return
		}
	}
}

// stop sends a signal to a running worked causing them to shut down.
func (w *worker) stop() {
	w.quit <- true
}
