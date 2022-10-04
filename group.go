package singleflight

import "sync"

type result[V any] struct {
	v   V
	err error
}

type Group[V any] struct {
	flightInterest map[string][]chan<- result[V]
	once           sync.Once
	mux            sync.Mutex
}

func (g *Group[V]) init() {
	g.once.Do(func() {
		g.flightInterest = make(map[string][]chan<- result[V])
	})
}

func (g *Group[V]) Do(key string, f func() (V, error)) (V, error) {
	g.init()

	interestChan := make(chan result[V], 1)

	// register interest
	g.mux.Lock()
	flightInterests := g.flightInterest[key]
	isFirstInterest := len(flightInterests) == 0
	flightInterests = append(flightInterests, interestChan)
	g.flightInterest[key] = flightInterests
	g.mux.Unlock()

	if isFirstInterest {
		go func() {
			v, err := f()
			r := result[V]{
				v:   v,
				err: err,
			}
			g.mux.Lock()
			interests := g.flightInterest[key]
			delete(g.flightInterest, key)
			g.mux.Unlock()
			for _, ch := range interests {
				ch <- r
			}
		}()
	}

	res := <-interestChan
	return res.v, res.err
}
