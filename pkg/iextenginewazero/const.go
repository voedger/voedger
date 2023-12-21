/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package iextenginewasm

const bitsInFourBytes = 32
const keysCapacity = 10
const keysBuildersCapacity = 10
const valuesCapacity = 10
const valueBuildersCapacity = 10

const (
	maxMemoryPages = 0xffff

	WasmPreallocatedBufferIncrease = 1000
)

var WasmPreallocatedBufferSize uint32 = 1000000
