/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package pool_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appparts/internal/pool"
)

func Example() {
	// This example demonstrates how to use Pool.

	// Create a new pool of integers.
	p := pool.New[int]([]int{1, 2, 3})

	func() {
		fmt.Println("Simple borrow-use-release")

		// Borrow a value from pool.
		v, err := p.Borrow()
		if err != nil {
			panic(err)
		}

		// Use value.
		fmt.Println(v, "enough:", p.Len())

		// Release the value to pool.
		p.Release(v)
	}()

	func() {
		fmt.Println("Borrow values until get error, then releases all borrowed")
		all := make([]int, 0, 3)

		// Borrow all values from pool.
		for v, err := p.Borrow(); err == nil; v, err = p.Borrow() {
			fmt.Println(v, "enough:", p.Len())
			all = append(all, v)
		}

		// Release borrowed values to pool.
		for _, v := range all {
			p.Release(v)
		}
	}()

	// Output:
	// Simple borrow-use-release
	// 1 enough: 2
	// Borrow values until get error, then releases all borrowed
	// 2 enough: 2
	// 3 enough: 1
	// 1 enough: 0
}
