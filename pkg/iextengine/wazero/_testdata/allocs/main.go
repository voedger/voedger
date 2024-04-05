/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package main

import (
	ext "github.com/voedger/voedger/pkg/exttinygo"
)

/*
Memory allocation & garbace collector tests.
See also:
	- https://tinygo.org/docs/concepts/compiler-internals/heap-allocation/
	- https://pkg.go.dev/github.com/tinygo-org/tinygo/src/runtime#MemStats
*/

var arr []*int32

//export arrAppend
func arrAppend() {
	event := ext.MustGetValue(ext.KeyBuilder(ext.StorageEvent, ext.NullEntity))
	n := event.AsInt32("offs")
	arr = append(arr, &n)
}

//export arrReset
func arrReset() {
	arr = make([]*int32, 0)
}

//export longFunc
func longFunc() {
	m := make(map[int]int)
	for i := 0; i < 10000000; i++ {
		if _, ok := m[i]; !ok {
			m[i] = i
		}
	}
}

//export simple
func simple() {
}

//export arrAppend2
func arrAppend2() {
	for i := 0; i < 10000; i++ {
		n := int32(i)
		arr = append(arr, &n)
	}
}

func main() {
}
