package singleflight

import "sync"

type result[V any] struct {
	v   V
	err error
}

type Group[V any] struct {
	interests []chan<- result[V]
	mux       sync.Mutex
}

func (g *Group[V]) Do(f func() (V, error)) (V, error) {
	interestChan := make(chan result[V], 1)

	// register interest
	g.mux.Lock()
	isFirstInterest := len(g.interests) == 0
	g.interests = append(g.interests, interestChan)
	g.mux.Unlock()

	if isFirstInterest {
		go func() {
			r := resultOf(f)
			g.mux.Lock()
			interests := g.interests
			g.interests = make([]chan<- result[V], 0, 64)
			g.mux.Unlock()
			for _, ch := range interests {
				ch <- r
			}
		}()
	}

	res := <-interestChan
	return res.v, res.err
}

func resultOf[V any](f func() (V, error)) result[V] {
	v, err := f()
	return result[V]{
		v:   v,
		err: err,
	}
}
