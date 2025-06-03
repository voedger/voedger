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

// Write int8 to buf
func WriteInt8(buf *bytes.Buffer, value int8) {
	buf.Write([]byte{byte(value)})
}

// Write byte to buf
func WriteByte(buf *bytes.Buffer, value byte) {
	buf.Write([]byte{value})
}

// Write bool to buf
func WriteBool(buf *bytes.Buffer, value bool) {
	s := []byte{0}
	if value {
		s[0] = 1
	}
	buf.Write(s)
}

// Write int16 to buf
func WriteInt16(buf *bytes.Buffer, value int16) {
	s := []byte{0, 0}
	BigEndianPutInt16(s, value)
	buf.Write(s)
}

// Write uint16 to buf
func WriteUint16(buf *bytes.Buffer, value uint16) {
	s := []byte{0, 0}
	binary.BigEndian.PutUint16(s, value)
	buf.Write(s)
}

// Write int32 to buf
func WriteInt32(buf *bytes.Buffer, value int32) {
	s := []byte{0, 0, 0, 0}
	BigEndianPutInt32(s, value)
	buf.Write(s)
}

// Write uint32 to buf
func WriteUint32(buf *bytes.Buffer, value uint32) {
	s := []byte{0, 0, 0, 0}
	binary.BigEndian.PutUint32(s, value)
	buf.Write(s)
}

// Write int64 to buf
func WriteInt64(buf *bytes.Buffer, value int64) {
	s := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	BigEndianPutInt64(s, value)
	buf.Write(s)
}

// Write uint64 to buf
func WriteUint64(buf *bytes.Buffer, value uint64) {
	s := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	binary.BigEndian.PutUint64(s, value)
	buf.Write(s)
}

// Write float32 to buf
func WriteFloat32(buf *bytes.Buffer, value float32) {
	s := []byte{0, 0, 0, 0}

	binary.BigEndian.PutUint32(s, math.Float32bits(value))
	buf.Write(s)
}

// Write float64 to buf
func WriteFloat64(buf *bytes.Buffer, value float64) {
	s := []byte{0, 0, 0, 0, 0, 0, 0, 0}

	binary.BigEndian.PutUint64(s, math.Float64bits(value))
	buf.Write(s)
}

// Writes short (< 64K) string into a buffer
func WriteShortString(buf *bytes.Buffer, str string) {
	const maxLen uint16 = 0xFFFF

	var l uint16
	line := str

	if len(line) < int(maxLen) {
		l = uint16(len(line)) //nolint G115
	} else {
		l = maxLen
		line = line[0:maxLen]
	}

	WriteUint16(buf, l)
	buf.WriteString(line)
}

// Writes data to buffer.
//
// # Hints:
//   - To exclude slow write through binary.Write() cast user types to go-types as possible.
//   - To avoid escaping values to heap during interface conversion use Write××× routines as possible.
func SafeWriteBuf(b *bytes.Buffer, data any) {
	var err error
	switch v := data.(type) {
	case nil:
	case int8:
		WriteInt8(b, v)
	case uint8:
		WriteByte(b, v)
	case int16:
		WriteInt16(b, v)
	case uint16:
		WriteUint16(b, v)
	case int32:
		WriteInt32(b, v)
	case uint32:
		WriteUint32(b, v)
	case int64:
		WriteInt64(b, v)
	case uint64:
		WriteUint64(b, v)
	case bool:
		WriteBool(b, v)
	case float32:
		WriteFloat32(b, v)
	case float64:
		WriteFloat64(b, v)
	case []byte:
		_, err = b.Write(v)
	case string:
		_, err = b.WriteString(v)
	default:
		err = binary.Write(b, binary.BigEndian, v)
	}
	if err != nil {
		panic(err)
	}
}

// Returns error if buf shorter than len bytes
func checkBufLen(buf *bytes.Buffer, length int) error {
	if l := buf.Len(); l < length {
		return fmt.Errorf("error read data from byte buffer, expected %d bytes, but only %d bytes is available: %w", length, l, io.ErrUnexpectedEOF)
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
	return BigEndianInt16(buf.Next(size)), nil
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
	return BigEndianInt32(buf.Next(size)), nil
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
	return BigEndianInt64(buf.Next(size)), nil
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

// Returns a slice of bytes built from the specified values, written from left to right
func ToBytes(value ...any) []byte {
	buf := new(bytes.Buffer)
	for _, p := range value {
		SafeWriteBuf(buf, p)
	}
	return buf.Bytes()
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
			next[i]++
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

// nolint revive
func BigEndianInt32(b []byte) int32 {
	_ = b[3] // bounds check
	return int32(b[3]) | int32(b[2])<<8 | int32(b[1])<<16 | int32(b[0])<<24
}

// nolint revive
func BigEndianPutInt16(b []byte, v int16) {
	_ = b[1] // bounds check
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}

// nolint revive
func BigEndianPutInt32(b []byte, v int32) {
	_ = b[3] // bounds check
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}

// nolint revive
func BigEndianPutInt64(b []byte, v int64) {
	_ = b[7] // bounds check
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
}

// nolint revive
func BigEndianInt16(b []byte) int16 {
	_ = b[1] // bounds check
	return int16(b[1]) | int16(b[0])<<8
}

// nolint revive
func BigEndianInt64(b []byte) int64 {
	_ = b[7] // bounds check
	return int64(b[7]) | int64(b[6])<<8 | int64(b[5])<<16 | int64(b[4])<<24 |
		int64(b[3])<<32 | int64(b[2])<<40 | int64(b[1])<<48 | int64(b[0])<<56
}
