/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package main

import "math"

//export Sin
func Sin(i1, i2 uint64) {
	_ = math.Sin(float64(i1 / i2))
}

//export StupidPow
func StupidPow(base, pow uint64) (res uint64) {
	res = 1
	for i := uint64(0); i < pow; i++ {
		res *= base
		if res == 0 {
			res = 1
		}
	}
	return res
}

// func Pow(base, pow uint64) uint64 {
// 	if base == 0 {
// 		return 0 // 0 raised to any power is 0
// 	}
// 	result := uint64(1)
// 	for pow > 0 {
// 		if pow&1 == 1 {
// 			result *= base
// 		}
// 		base *= base
// 		pow >>= 1
// 	}
// 	return result
// }
