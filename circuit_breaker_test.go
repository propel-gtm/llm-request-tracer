package llmtracer

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker(t *testing.T) {
	t.Run("starts in closed state", func(t *testing.T) {
		cb := NewCircuitBreaker(3, 100*time.Millisecond)
		assert.Equal(t, StateClosed, cb.GetState())
		assert.False(t, cb.IsOpen())
	})

	t.Run("opens after max failures", func(t *testing.T) {
		cb := NewCircuitBreaker(3, 100*time.Millisecond)
		testErr := errors.New("test error")

		// First two failures - should remain closed
		for i := 0; i < 2; i++ {
			err := cb.Call(func() error { return testErr })
			assert.Equal(t, testErr, err)
			assert.Equal(t, StateClosed, cb.GetState())
		}

		// Third failure - should open
		err := cb.Call(func() error { return testErr })
		assert.Equal(t, testErr, err)
		assert.Equal(t, StateOpen, cb.GetState())
		assert.True(t, cb.IsOpen())
	})

	t.Run("fails fast when open", func(t *testing.T) {
		cb := NewCircuitBreaker(1, 100*time.Millisecond)

		// Open the circuit
		cb.Call(func() error { return errors.New("error") })
		assert.Equal(t, StateOpen, cb.GetState())

		// Should fail fast without calling the function
		called := false
		err := cb.Call(func() error {
			called = true
			return nil
		})

		assert.Equal(t, ErrCircuitOpen, err)
		assert.False(t, called)
	})

	t.Run("transitions to half-open after timeout", func(t *testing.T) {
		cb := NewCircuitBreaker(1, 50*time.Millisecond)

		// Open the circuit
		cb.Call(func() error { return errors.New("error") })
		assert.Equal(t, StateOpen, cb.GetState())

		// Wait for timeout
		time.Sleep(60 * time.Millisecond)

		// Should be half-open now
		assert.Equal(t, StateHalfOpen, cb.GetState())

		// Call should be attempted
		called := false
		err := cb.Call(func() error {
			called = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("closes after successful calls in half-open", func(t *testing.T) {
		cb := NewCircuitBreaker(1, 50*time.Millisecond)

		// Open the circuit
		cb.Call(func() error { return errors.New("error") })

		// Wait for timeout
		time.Sleep(60 * time.Millisecond)

		// First successful call in half-open
		err := cb.Call(func() error { return nil })
		assert.NoError(t, err)
		assert.Equal(t, StateHalfOpen, cb.GetState())

		// Second successful call should close the circuit
		err = cb.Call(func() error { return nil })
		assert.NoError(t, err)
		assert.Equal(t, StateClosed, cb.GetState())
	})

	t.Run("reopens on failure in half-open", func(t *testing.T) {
		cb := NewCircuitBreaker(1, 50*time.Millisecond)

		// Open the circuit
		cb.Call(func() error { return errors.New("error") })

		// Wait for timeout
		time.Sleep(60 * time.Millisecond)

		// Failure in half-open should reopen
		err := cb.Call(func() error { return errors.New("still failing") })
		assert.Error(t, err)
		assert.Equal(t, StateOpen, cb.GetState())
	})

	t.Run("resets failure count on success in closed state", func(t *testing.T) {
		cb := NewCircuitBreaker(3, 100*time.Millisecond)

		// Two failures
		cb.Call(func() error { return errors.New("error") })
		cb.Call(func() error { return errors.New("error") })

		// Success should reset
		err := cb.Call(func() error { return nil })
		assert.NoError(t, err)

		// Two more failures should not open (count was reset)
		cb.Call(func() error { return errors.New("error") })
		cb.Call(func() error { return errors.New("error") })
		assert.Equal(t, StateClosed, cb.GetState())

		// Third failure opens
		cb.Call(func() error { return errors.New("error") })
		assert.Equal(t, StateOpen, cb.GetState())
	})

	t.Run("concurrent access", func(t *testing.T) {
		cb := NewCircuitBreaker(10, 100*time.Millisecond)
		var wg sync.WaitGroup

		// Run many concurrent operations
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				if i%2 == 0 {
					cb.Call(func() error { return nil })
				} else {
					cb.Call(func() error { return errors.New("error") })
				}
			}(i)
		}

		wg.Wait()
		// Should not panic or deadlock
	})
}
