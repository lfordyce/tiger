package domain

type Sharder interface {
	Shard(Request) map[string]uint64
}

type SharderFunc func(Request) map[string]uint64

func (sf SharderFunc) Shard(r Request) map[string]uint64 {
	return sf(r)
}

type MultiShardHandler struct {
	sharder         Sharder
	shardHandlers   map[string]ShardHandler
	unknownHandlers []ShardHandler
}

type MultiShardHandlerOpt func(*MultiShardHandler)

func WithShardHandler(shard string, h ShardHandler) MultiShardHandlerOpt {
	return MultiShardHandlerOpt(func(msh *MultiShardHandler) {
		msh.shardHandlers[shard] = h
	})
}

func MultiSharder(sharder Sharder, opts ...MultiShardHandlerOpt) *MultiShardHandler {
	msh := &MultiShardHandler{
		sharder:         sharder,
		shardHandlers:   make(map[string]ShardHandler),
		unknownHandlers: make([]ShardHandler, 0),
	}
	for _, o := range opts {
		o(msh)
	}
	return msh
}

func (msf *MultiShardHandler) Process(r Request) error {
	for shard, count := range msf.sharder.Shard(r) {
		if h, ok := msf.shardHandlers[shard]; ok {
			if count > 0 {
				h := h
				r := r
				n := count
				go h.Process(r, int(n))
			}
		} else {
			for _, h := range msf.unknownHandlers {
				h := h
				r := r
				n := count
				go h.Process(r, int(n))
			}
		}
	}
	return nil
}
