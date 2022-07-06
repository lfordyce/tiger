package queue

type Dispatcher interface {
	Run()
	Queue(Job)
	Stop()
}

type dispatcher struct {
	name       string
	jobQueue   chan Job
	workers    []*worker
	workerPool chan chan Job

	quit chan bool
}

func NewDispatcher(name string, workers int) *dispatcher {
	d := &dispatcher{
		name:       name,
		jobQueue:   make(chan Job),
		workers:    make([]*worker, workers),
		workerPool: make(chan chan Job, workers),
		quit:       make(chan bool, 1),
	}

	for i := 0; i < workers; i++ {
		d.workers[i] = newWorker(d, i)
	}

	return d
}

// Run starts the queue and loops to process all incoming requests. When a job
// is received, a worker will attempt to be dequeued from the worker pool,
func (d *dispatcher) Run() {
	for _, w := range d.workers {
		go w.run()
	}

	for {
		select {
		case j := <-d.jobQueue:
			// new job has been received by the dispatcher
			go func(j Job) {
				// dispatch the job to the worker pool
				<-d.workerPool <- j
			}(j)
		case <-d.quit:
			for _, w := range d.workers {
				w.stop()
			}
			return
		}
	}
}

// Queue a new job of the dispatcher to submit to the worker pool
func (d *dispatcher) Queue(j Job) {
	d.jobQueue <- j
}

// Stop signals the works to stop handling new job requests
func (d *dispatcher) Stop() {
	d.quit <- true
}
