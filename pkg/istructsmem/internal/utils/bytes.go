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

// Reads short (< 64K) string from a buffer
func ReadShortString(buf *bytes.Buffer) (string, error) {
	var strLen uint16
	if err := binary.Read(buf, binary.BigEndian, &strLen); err != nil {
		return "", fmt.Errorf("error read string length: %w", err)
	}
	if strLen == 0 {
		return "", nil
	}
	if buf.Len() < int(strLen) {
		return "", fmt.Errorf("error read string, expected %d bytes, but only %d bytes is available: %w", strLen, buf.Len(), io.ErrUnexpectedEOF)
	}
	return string(buf.Next(int(strLen))), nil
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

// ToBytes returns bytes slice constructed from specified values writed from left to right
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
func SuccBytes(cur []byte) (next []byte) {
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
