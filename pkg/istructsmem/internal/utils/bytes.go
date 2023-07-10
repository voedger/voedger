/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// Copies bytes from src
func CopyBytes(src []byte) []byte {
	result := make([]byte, len(src))
	copy(result, src)
	return result
}

// Writes value to bytes buffer.
//
// # Panics:
//   - if any buffer write error
func SafeWriteBuf(b *bytes.Buffer, data any) {
	var err error
	switch v := data.(type) {
	case nil:
	case []byte:
		_, err = b.Write(v)
	case string:
		_, err = b.WriteString(v)
	default:
		err = binary.Write(b, binary.BigEndian, v)
	}
	if err != nil {
		// notest: Difficult to get an error when writing to bytes.buffer
		panic(err)
	}
}

// Writes short (< 64K) string into a buffer
func WriteShortString(buf *bytes.Buffer, str string) {
	const maxLen uint16 = 0xFFFF

	var l uint16
	line := str

	if len(line) < int(maxLen) {
		l = uint16(len(line))
	} else {
		l = maxLen
		line = line[0:maxLen]
	}

	SafeWriteBuf(buf, l)
	SafeWriteBuf(buf, line)
}

// Returns error if buf shorter than len bytes
func checkBufLen(buf *bytes.Buffer, len int) error {
	if l := buf.Len(); l < len {
		return fmt.Errorf("error read data from byte buffer, expected %d bytes, but only %d bytes is available: %w", len, l, io.ErrUnexpectedEOF)
	}
	return nil
}

// Reads int8 from buf
func ReadInt8(buf *bytes.Buffer) (int8, error) {
	i, e := buf.ReadByte()
	if e == io.EOF {
		e = io.ErrUnexpectedEOF
	}
	return int8(i), e
}

// Reads byte from buf
func ReadByte(buf *bytes.Buffer) (byte, error) {
	i, e := buf.ReadByte()
	if e == io.EOF {
		e = io.ErrUnexpectedEOF
	}
	return i, e
}

// Reads bool from buf
func ReadBool(buf *bytes.Buffer) (bool, error) {
	i, e := ReadByte(buf)
	return i != 0, e
}

// Reads int16 from buf
func ReadInt16(buf *bytes.Buffer) (int16, error) {
	const size = 2
	if err := checkBufLen(buf, size); err != nil {
		return 0, err
	}
	return int16(binary.BigEndian.Uint16(buf.Next(size))), nil
}

// Reads uint16 from buf
func ReadUInt16(buf *bytes.Buffer) (uint16, error) {
	const size = 2
	if err := checkBufLen(buf, size); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(buf.Next(size)), nil
}

// Reads int32 from buf
func ReadInt32(buf *bytes.Buffer) (int32, error) {
	const size = 4
	if err := checkBufLen(buf, size); err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(buf.Next(size))), nil
}

// Reads uint32 from buf
func ReadUInt32(buf *bytes.Buffer) (uint32, error) {
	const size = 4
	if err := checkBufLen(buf, size); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(buf.Next(size)), nil
}

// Reads int64 from buf
func ReadInt64(buf *bytes.Buffer) (int64, error) {
	const size = 8
	if err := checkBufLen(buf, size); err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(buf.Next(size))), nil
}

// Reads uint64 from buf
func ReadUInt64(buf *bytes.Buffer) (uint64, error) {
	const size = 8
	if err := checkBufLen(buf, size); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(buf.Next(size)), nil
}

// Reads float32 from buf
func ReadFloat32(buf *bytes.Buffer) (float32, error) {
	const size = 4
	if err := checkBufLen(buf, size); err != nil {
		return 0, err
	}
	return math.Float32frombits(binary.BigEndian.Uint32(buf.Next(size))), nil
}

// Reads float64 from buf
func ReadFloat64(buf *bytes.Buffer) (float64, error) {
	const size = 8
	if err := checkBufLen(buf, size); err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.BigEndian.Uint64(buf.Next(size))), nil
}

// Reads short (< 64K) string from a buffer
func ReadShortString(buf *bytes.Buffer) (string, error) {
	const size = 2
	if err := checkBufLen(buf, size); err != nil {
		return "", err
	}
	strLen := int(binary.BigEndian.Uint16(buf.Next(size)))
	if strLen == 0 {
		return "", nil
	}
	if err := checkBufLen(buf, strLen); err != nil {
		return "", err
	}
	return string(buf.Next(strLen)), nil
}

// Expands (from left) value by write specified prefixes
func PrefixBytes(value []byte, prefix ...interface{}) []byte {
	buf := new(bytes.Buffer)
	for _, p := range prefix {
		SafeWriteBuf(buf, p)
	}
	if len(value) > 0 {
		SafeWriteBuf(buf, value)
	}
	return buf.Bytes()
}

// Returns a slice of bytes built from the specified values, written from left to right
func ToBytes(value ...interface{}) []byte {
	return PrefixBytes(nil, value...)
}

// Returns is all bytes is max (0xFF)
func FullBytes(b []byte) bool {
	for _, e := range b {
		if e != math.MaxUint8 {
			return false
		}
	}
	return true
}

// Increments by one bit low byte and returns the result.
//
// Useful to obtain right margin of half-open range of partially filled clustering columns
func IncBytes(cur []byte) (next []byte) {
	if FullBytes(cur) {
		return nil
	}

	var incByte func(i int)
	incByte = func(i int) {
		if next[i] != math.MaxUint8 {
			next[i] = next[i] + 1
			return
		}
		next[i] = 0
		incByte(i - 1)
	}
	next = make([]byte, len(cur))
	copy(next, cur)
	incByte(len(cur) - 1)
	return next
}
