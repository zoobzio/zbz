package zlog

import (
	"io"
	"slices"
	"sync"
)

// writer implements io.Writer and pipes output to registered writers
type writer struct {
	original io.Writer
	writers  []io.Writer
	mu       sync.RWMutex
}

// Write implements io.Writer interface
func (w *writer) Write(p []byte) (n int, err error) {
	// Always write to original first (never break logging)
	n, err = w.original.Write(p)

	// Then pipe to writers (best effort, async)
	w.mu.RLock()
	writers := make([]io.Writer, len(w.writers))
	copy(writers, w.writers)
	w.mu.RUnlock()

	// Execute writes asynchronously so they don't block logging
	for _, writer := range writers {
		go func(wr io.Writer, data []byte) {
			defer func() {
				if r := recover(); r != nil {
					// Writer panicked - don't break logging
				}
			}()
			_, _ = wr.Write(data) // Ignore errors - don't break logging
		}(writer, slices.Clone(p)) // Copy bytes
	}

	return n, err
}
