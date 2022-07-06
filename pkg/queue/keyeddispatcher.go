package queue

import "sync"

//type KeyedDispatcher[J Job, K comparable] interface {
//	Dispatcher
//	DispatchToKey(J, K)
//}

type KeyedDispatcher interface {
	Dispatcher
	DispatchToKey(job Job, key string)
}

type keyedWorker struct {
	id       int
	key      string
	jobQueue chan Job
	quit     chan bool
}

type keyedDispatcher struct {
	name       string
	jobQueue   chan Job
	mapLock    sync.Mutex
	workerMap  map[string]*keyedWorker
	workerPool chan chan Job

	quit chan bool
}

func NewKeyedDispatcher(name string, workers int) *keyedDispatcher {
	d := &keyedDispatcher{
		name:       name,
		jobQueue:   make(chan Job),
		workerMap:  make(map[string]*keyedWorker),
		workerPool: make(chan chan Job, workers),
		quit:       make(chan bool, 1),
	}
	return d
}
