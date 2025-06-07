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

func Benchmark_BufWriteCustomTypes(b *testing.B) {

	type (
		synType = uint64
		usrType uint64
	)

	const capacity = 1000 // test slices capacity

	int64slice := make([]uint64, capacity)
	for i := range int64slice {
		int64slice[i] = uint64(2 * i)
	}

	usrSlice := make([]usrType, capacity)
	for i := range usrSlice {
		usrSlice[i] = usrType(2 * i)
	}

	synSlice := make([]synType, capacity)
	for i := range synSlice {
		synSlice[i] = synType(2 * i)
	}

	var int64bytes []byte
	b.Run("1. SafeWriteBuf go-type", func(b *testing.B) {
		for b.Loop() {
			buf := bytes.NewBuffer(nil)
			for i := range int64slice {
				SafeWriteBuf(buf, int64slice[i])
			}
			if len(int64bytes) == 0 {
				int64bytes = buf.Bytes()
			}
		}
	})

	var usrBytes []byte
	b.Run("2. SafeWriteBuf user type", func(b *testing.B) {
		for b.Loop() {
			buf := bytes.NewBuffer(nil)
			for i := range usrSlice {
				SafeWriteBuf(buf, usrSlice[i])
			}
			if len(usrBytes) == 0 {
				usrBytes = buf.Bytes()
			}
		}
	})

	var synBytes []byte
	b.Run("3. SafeWriteBuf synonym type", func(b *testing.B) {
		for b.Loop() {
			buf := bytes.NewBuffer(nil)
			for i := range synSlice {
				SafeWriteBuf(buf, synSlice[i])
			}
			if len(synBytes) == 0 {
				synBytes = buf.Bytes()
			}
		}
	})

	var castBytes []byte
	b.Run("4. SafeWriteBuf cast user type", func(b *testing.B) {
		for b.Loop() {
			buf := bytes.NewBuffer(nil)
			for i := range usrSlice {
				SafeWriteBuf(buf, uint64(usrSlice[i]))
			}
			if len(castBytes) == 0 {
				castBytes = buf.Bytes()
			}
		}
	})

	require := require.New(b)
	require.EqualValues(int64bytes, usrBytes)
	require.EqualValues(int64bytes, synBytes)
	require.EqualValues(int64bytes, castBytes)

	var b_ []byte
	b.Run("5. WriteUint64, no heap escapes", func(b *testing.B) {
		for b.Loop() {
			buf := bytes.NewBuffer(nil)
			for i := range int64slice {
				WriteUint64(buf, int64slice[i])
			}
			if len(b_) == 0 {
				b_ = buf.Bytes()
			}
		}
	})

	require.EqualValues(int64bytes, b_)
}

// Writes value to bytes buffer.
//
// Deprecated. For benchmark test purposes only
func old_SafeWriteBuf(b *bytes.Buffer, data any) {
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

/*
2023-07-15, Nikolay Nikitin, go 1.20.6

Running tool: D:\Go\bin\go.exe test -benchmem -run=^$ -bench ^Benchmark_BufWrite$ github.com/voedger/voedger/pkg/istructsmem/internal/utils

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/istructsmem/internal/utils
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
Benchmark_BufWrite/old_SafeWriteBuf_via_buf.Write-4         	 1734220	       667.9 ns/op	     192 B/op	      18 allocs/op
Benchmark_BufWrite/new_SafeWriteBuf_via_Write×××-4          	 3469579	       347.4 ns/op	     144 B/op	       7 allocs/op
Benchmark_BufWrite/naked_Write×××-4                         	 9676279	       125.8 ns/op	      64 B/op	       1 allocs/op
PASS
ok  	github.com/voedger/voedger/pkg/istructsmem/internal/utils	4.899s
*/

/*
2025-05-30, Nikolay Nikitin, go 1.24.2

Running tool: C:\Program Files\Go\bin\go.exe test -benchmem -run=^$ -bench ^Benchmark_BufWrite$ github.com/voedger/voedger/pkg/istructsmem/internal/utils -count=1 -v

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/istructsmem/internal/utils
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
Benchmark_BufWrite/old_SafeWriteBuf_via_buf.Write-4             1726729	       688.6 ns/op	     160 B/op	      13 allocs/op
Benchmark_BufWrite/new_SafeWriteBuf_via_Write×××-4              4626688	       254.0 ns/op	     112 B/op	       2 allocs/op
Benchmark_BufWrite/naked_Write×××-4                             6287666	       191.1 ns/op	     112 B/op	       2 allocs/op
PASS
ok  	github.com/voedger/voedger/pkg/istructsmem/internal/utils	3.623s
*/

func Benchmark_BufWrite(b *testing.B) {

	type s struct {
		int8
		int16
		int32
		int64
		uint8
		uint16
		uint32
		uint64
		bool
		float32
		float64
	}

	s1 := s{-8, -16, -32, -64, 8, 16, 32, 64, true, 3.14159265358, 3.141592653589793238}

	buf := bytes.NewBuffer(nil)
	old_SafeWriteBuf(buf, s1.int8)
	old_SafeWriteBuf(buf, s1.int16)
	old_SafeWriteBuf(buf, s1.int32)
	old_SafeWriteBuf(buf, s1.int64)
	old_SafeWriteBuf(buf, s1.uint8)
	old_SafeWriteBuf(buf, s1.uint16)
	old_SafeWriteBuf(buf, s1.uint32)
	old_SafeWriteBuf(buf, s1.uint64)
	old_SafeWriteBuf(buf, s1.bool)
	old_SafeWriteBuf(buf, s1.float32)
	old_SafeWriteBuf(buf, s1.float64)

	data := buf.Bytes()

	b.Run("old SafeWriteBuf via buf.Write", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			buf := bytes.NewBuffer(nil)
			old_SafeWriteBuf(buf, s1.int8)
			old_SafeWriteBuf(buf, s1.int16)
			old_SafeWriteBuf(buf, s1.int32)
			old_SafeWriteBuf(buf, s1.int64)
			old_SafeWriteBuf(buf, s1.uint8)
			old_SafeWriteBuf(buf, s1.uint16)
			old_SafeWriteBuf(buf, s1.uint32)
			old_SafeWriteBuf(buf, s1.uint64)
			old_SafeWriteBuf(buf, s1.bool)
			old_SafeWriteBuf(buf, s1.float32)
			old_SafeWriteBuf(buf, s1.float64)

			if i == 0 {
				require.New(b).EqualValues(data, buf.Bytes())
			}
		}
	})

	b.Run("new SafeWriteBuf via Write×××", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			buf := bytes.NewBuffer(nil)
			SafeWriteBuf(buf, s1.int8)
			SafeWriteBuf(buf, s1.int16)
			SafeWriteBuf(buf, s1.int32)
			SafeWriteBuf(buf, s1.int64)
			SafeWriteBuf(buf, s1.uint8)
			SafeWriteBuf(buf, s1.uint16)
			SafeWriteBuf(buf, s1.uint32)
			SafeWriteBuf(buf, s1.uint64)
			SafeWriteBuf(buf, s1.bool)
			SafeWriteBuf(buf, s1.float32)
			SafeWriteBuf(buf, s1.float64)

			if i == 0 {
				require.New(b).EqualValues(data, buf.Bytes())
			}
		}
	})

	b.Run("naked Write×××", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			buf := bytes.NewBuffer(nil)
			WriteInt8(buf, s1.int8)
			WriteInt16(buf, s1.int16)
			WriteInt32(buf, s1.int32)
			WriteInt64(buf, s1.int64)
			WriteByte(buf, s1.uint8)
			WriteUint16(buf, s1.uint16)
			WriteUint32(buf, s1.uint32)
			WriteUint64(buf, s1.uint64)
			WriteBool(buf, s1.bool)
			WriteFloat32(buf, s1.float32)
			WriteFloat64(buf, s1.float64)

			if i == 0 {
				require.New(b).EqualValues(data, buf.Bytes())
			}
		}
	})
}

/*
2023-07-15, Nikolay Nikitin, go 1.20.6

Running tool: D:\Go\bin\go.exe test -benchmem -run=^$ -bench ^Benchmark_BufRead$ github.com/voedger/voedger/pkg/istructsmem/internal/utils

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/istructsmem/internal/utils
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
Benchmark_BufRead/buf.Read-4         	 2714758	       479.5 ns/op	     128 B/op	      10 allocs/op
Benchmark_BufRead/Read×××-4          	29999174	        46.52 ns/op	       0 B/op	       0 allocs/op
Benchmark_BufRead/no_err_control-4   	41306666	        28.52 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/voedger/voedger/pkg/istructsmem/internal/utils	4.473s
*/

/*
2025-05-30, Nikolay Nikitin, go 1.24.2

Running tool: C:\Program Files\Go\bin\go.exe test -benchmem -run=^$ -bench ^Benchmark_BufRead$ github.com/voedger/voedger/pkg/istructsmem/internal/utils -count=1 -v

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/istructsmem/internal/utils
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
Benchmark_BufRead/buf.Read-4            2615833	       493.4 ns/op	      80 B/op	       9 allocs/op
Benchmark_BufRead/Read×××-4            12698040	        94.35 ns/op	      48 B/op	       1 allocs/op
Benchmark_BufRead/no_err_control-4     13905964	        78.68 ns/op	      48 B/op	       1 allocs/op
PASS
ok  	github.com/voedger/voedger/pkg/istructsmem/internal/utils	3.644s
*/

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
		for i := 0; b.Loop(); i++ {
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
		for i := 0; b.Loop(); i++ {
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
		for i := 0; b.Loop(); i++ {
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
