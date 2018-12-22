package dataloader

import (
	"sync"
	"time"
)

func NewDataLoader(maxBatch int, wait time.Duration, fetch Fetcher) *DataLoader {
	return &DataLoader{
		maxBatch: maxBatch,
		wait:     wait,
		fetch:    fetch,
	}
}

// Key concept by facebook's data loader https://github.com/facebook/dataloader.
// Golang implementation inspired by https://github.com/vektah/dataloaden.
type DataLoader struct {
	// this method provides the data for the loader
	fetch Fetcher

	// how long to done before sending a batch
	wait time.Duration

	// this will limit the maximum number of keys to send in one batch, 0 = no limit
	maxBatch int

	// INTERNAL

	// lazily created cache
	cache map[Key]Value

	// the current batch. keys will continue to be collected until timeout is hit,
	// then everything will be sent to the fetch method and out to the listeners
	batch *batch

	// mutex to prevent races
	mu sync.Mutex
}

type Key interface{}
type Value interface{}

type Fetcher func(keys []Key) ([]Value, []error)

type batch struct {
	// batched keys collected until batch timeout
	keys    []Key
	data    []Value
	error   []error
	closing bool
	done    chan struct{}
}

// Load a user by key, batching and caching will be applied automatically
func (l *DataLoader) Load(key Key) (Value, error) {
	return l.LoadThunk(key)()
}

// LoadThunk returns a function that when called will block waiting for a user.
// This method should be used if you want one goroutine to make requests to many
// different data loaders without blocking until the thunk is called.
func (l *DataLoader) LoadThunk(key Key) func() (Value, error) {
	l.mu.Lock()
	if it, ok := l.cache[key]; ok {
		l.mu.Unlock()
		return func() (Value, error) {
			return it, nil
		}
	}
	if l.batch == nil {
		l.batch = &batch{done: make(chan struct{})}
	}
	batch := l.batch
	pos := batch.keyIndex(l, key)
	l.mu.Unlock()

	return func() (Value, error) {
		<-batch.done

		var data Value
		if pos < len(batch.data) {
			data = batch.data[pos]
		}

		var err error
		// its convenient to be able to return a single error for everything
		if len(batch.error) == 1 {
			err = batch.error[0]
		} else if batch.error != nil {
			err = batch.error[pos]
		}

		if err == nil {
			l.mu.Lock()
			l.unsafeSet(key, data)
			l.mu.Unlock()
		}

		return data, err
	}
}

// LoadAll fetches many keys at once. It will be broken into appropriate sized
// sub batches depending on how the loader is configured
func (l *DataLoader) LoadAll(keys []Key) ([]Value, []error) {
	results := make([]func() (Value, error), len(keys))

	for i, key := range keys {
		results[i] = l.LoadThunk(key)
	}

	values := make([]Value, len(keys))
	errors := make([]error, len(keys))
	for i, thunk := range results {
		values[i], errors[i] = thunk()
	}
	return values, errors
}

// Prime the cache with the provided key and value.
// If the key already exists, no change is made
// and false is returned. Returns true if forced.
// (To forcefully prime the cache, use forcePrime = true.)
func (l *DataLoader) Prime(key Key, value Value, forcePrime bool) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	primeIt := forcePrime

	if !primeIt {
		var found bool
		if _, found = l.cache[key]; !found {
			primeIt = true
		}
	}

	if primeIt {
		l.unsafeSet(key, value)
	}
	return primeIt
}

// Clear the value at key from the cache, if it exists
func (l *DataLoader) Clear(key Key) *DataLoader {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.cache, key)
	return l
}

func (l *DataLoader) unsafeSet(key Key, value Value) {
	if l.cache == nil {
		l.cache = map[Key]Value{}
	}
	l.cache[key] = value
}

// keyIndex will return the location of the key in the batch, if its not found
// it will add the key to the batch
func (b *batch) keyIndex(l *DataLoader, key Key) int {
	for i, existingKey := range b.keys {
		if key == existingKey {
			return i
		}
	}

	pos := len(b.keys)
	b.keys = append(b.keys, key)
	if pos == 0 {
		go b.startTimer(l)
	}

	if l.maxBatch != 0 && pos >= l.maxBatch-1 {
		if !b.closing {
			b.closing = true
			l.batch = nil
			go b.end(l)
		}
	}

	return pos
}

func (b *batch) startTimer(l *DataLoader) {
	time.Sleep(l.wait)
	l.mu.Lock()

	// we must have hit a batch limit and are already finalizing this batch
	if b.closing {
		l.mu.Unlock()
		return
	}

	l.batch = nil
	l.mu.Unlock()

	b.end(l)
}

func (b *batch) end(l *DataLoader) {
	b.data, b.error = l.fetch(b.keys)
	close(b.done)
}
