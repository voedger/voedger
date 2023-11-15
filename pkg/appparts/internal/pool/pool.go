/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package pool

// Thread-safe pool of reusable values.
//
// Value is borrowed from poll by calling Borrow() method.
// After using value, it must be returned to pool by calling Release() method.
//
//TODO: adds tests to check that pool is thread-safe
type Pool[T any] struct {
	c chan T
}

// Creates and returns a new instance of Pool. All passed to New() values are initially placed into the pool.
func New[T any](values []T) *Pool[T] {
	p := &Pool[T]{
		c: make(chan T, len(values)),
	}
	for _, v := range values {
		p.c <- v
	}
	return p
}

// Borrows a value from pool.
//
// If pool is empty, returns ErrNotEnough.
//
// Borrowed value must be returned to pool by calling Release() method.
func (p *Pool[T]) Borrow() (value T, err error) {
	select {
	case v := <-p.c:
		return v, nil
	default:
		return value, ErrNotEnough
	}
}

// Returns a value into pool.
func (p *Pool[T]) Release(value T) {
	p.c <- value
}

// Returns a number of enough values in pool.
func (p *Pool[T]) Len() int {
	return len(p.c)
}
