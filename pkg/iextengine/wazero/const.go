/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package iextenginewazero

const bitsInFourBytes = 32

const (
	maxMemoryPages = 0xffff

	WasmPreallocatedBufferIncrease = 1000
)

var WasmPreallocatedBufferSize uint32 = 1000000
