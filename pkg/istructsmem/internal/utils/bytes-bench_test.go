/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package utils

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func Benchmark_BufRead(b *testing.B) {

	type s struct {
		a int16
		b int32
		c int64
		e uint16
		f uint32
		g uint64
		h byte
		i bool
	}

	s1 := s{-1, -2, -3, 4, 5, 6, 234, true}

	buf := new(bytes.Buffer)
	SafeWriteBuf(buf, s1.a)
	SafeWriteBuf(buf, s1.b)
	SafeWriteBuf(buf, s1.c)
	SafeWriteBuf(buf, s1.e)
	SafeWriteBuf(buf, s1.f)
	SafeWriteBuf(buf, s1.g)
	SafeWriteBuf(buf, s1.h)
	SafeWriteBuf(buf, s1.i)

	data := buf.Bytes()

	b.Run("buf.Read", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s2 := s{}
			buf := bytes.NewBuffer(data)
			binary.Read(buf, binary.BigEndian, &s2.a)
			binary.Read(buf, binary.BigEndian, &s2.b)
			binary.Read(buf, binary.BigEndian, &s2.c)
			binary.Read(buf, binary.BigEndian, &s2.e)
			binary.Read(buf, binary.BigEndian, &s2.f)
			binary.Read(buf, binary.BigEndian, &s2.g)
			binary.Read(buf, binary.BigEndian, &s2.h)
			binary.Read(buf, binary.BigEndian, &s2.i)

			if i == 0 {
				require.New(b).EqualValues(s1, s2)
			}
		}
	})

	b.Run("Read×××", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s2 := s{}
			buf := bytes.NewBuffer(data)
			s2.a, _ = ReadInt16(buf)
			s2.b, _ = ReadInt32(buf)
			s2.c, _ = ReadInt64(buf)
			s2.e, _ = ReadUInt16(buf)
			s2.f, _ = ReadUInt32(buf)
			s2.g, _ = ReadUInt64(buf)
			s2.h, _ = ReadByte(buf)
			s2.i, _ = ReadBool(buf)

			if i == 0 {
				require.New(b).EqualValues(s1, s2)
			}
		}
	})

	b.Run("no err control", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s2 := s{}
			buf := bytes.NewBuffer(data)
			s2.a = int16(binary.BigEndian.Uint16(buf.Next(2)))
			s2.b = int32(binary.BigEndian.Uint32(buf.Next(4)))
			s2.c = int64(binary.BigEndian.Uint64(buf.Next(8)))
			s2.e = binary.BigEndian.Uint16(buf.Next(2))
			s2.f = binary.BigEndian.Uint32(buf.Next(4))
			s2.g = binary.BigEndian.Uint64(buf.Next(8))
			s2.h, _ = buf.ReadByte()
			if i, _ := buf.ReadByte(); i > 0 {
				s2.i = true
			}

			if i == 0 {
				require.New(b).EqualValues(s1, s2)
			}
		}
	})
}
