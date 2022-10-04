// Package singleflight provides a duplicate function call suppression
// mechanism. It's heavily inspired by
// golang.org/x/sync/singleflight.package/singleflight
package singleflight

import "sync"

type result[V any] struct {
	v   V
	err error
}

// Group represents a collection of threads interested in the results of the
// same workload.
type Group[V any] struct {
	interests []chan<- result[V]
	mux       sync.Mutex
}

// Do ensures at most one workload is running at any given time. Multiple calls
// to Do will wait for the results of a ongoing workload, rather than starting
// a new one. Once a workload finishes the same result is returned to all
// callers and subsequent calls to Do will start a new workload.
func (g *Group[V]) Do(workload func() (V, error)) (V, error) { //nolint:ireturn
	interestChan := make(chan result[V], 1)

	// register interest
	g.mux.Lock()
	isFirstInterest := len(g.interests) == 0
	g.interests = append(g.interests, interestChan)
	g.mux.Unlock()

	if isFirstInterest {
		go func() {
			res := resultOf(workload)

			g.mux.Lock()
			interests := g.interests
			g.interests = nil
			g.mux.Unlock()

			for _, ch := range interests {
				ch <- res
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
