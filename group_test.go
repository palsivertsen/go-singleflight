package singleflight_test

import (
	"crypto/sha256"
	"strconv"
	"sync"
	"testing"

	"palsivertsen/go-singleflight"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroup_Do_MultiThreadSingleCall(t *testing.T) {
	t.Parallel()

	const numTests = 100

	var counter int
	var counterMu sync.Mutex
	var waitGroup sync.WaitGroup
	var startGroup sync.WaitGroup

	var unit singleflight.Group[int]

	waitGroup.Add(numTests)
	startGroup.Add(numTests)

	for i := 0; i < numTests; i++ {
		go func() {
			defer waitGroup.Done()
			startGroup.Done()

			val, err := unit.Do("test group", func() (int, error) {
				startGroup.Wait() // Wait for all go routines to start.

				// make this func long running by doing something compute heavy
				for i := 0; i < 1000; i++ {
					h := sha256.New()
					_, _ = h.Write([]byte(strconv.Itoa(i)))
					_ = h.Sum(nil)
				}

				counterMu.Lock()
				defer counterMu.Unlock()
				counter++

				return counter, nil
			})

			require.NoError(t, err)
			assert.Equal(t, 1, val)
		}()
	}

	waitGroup.Wait()
	assert.Equal(t, 1, counter)
}

func TestGroup_Do_Sequential(t *testing.T) {
	t.Parallel()

	const numTests = 100

	var counter int
	var waitGroup sync.WaitGroup

	var unit singleflight.Group[int]

	waitGroup.Add(numTests)

	for i := 0; i < numTests; i++ {
		val, err := unit.Do("test group", func() (int, error) {
			defer waitGroup.Done()
			counter++

			return counter, nil
		})

		require.NoError(t, err)
		assert.Equal(t, i+1, val)
	}

	waitGroup.Wait()
	assert.Equal(t, numTests, counter)
}

type testError string

func (e testError) Error() string { return string(e) }

func TestGroup_Do_Error(t *testing.T) {
	t.Parallel()

	var unit singleflight.Group[int]

	val, err := unit.Do("test group", func() (int, error) {
		return 0, testError("test error")
	})

	assert.ErrorIs(t, err, testError("test error"))
	assert.Equal(t, 0, val)
}
