/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package utils

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSafeWriteBuf(t *testing.T) {
	type args struct {
		buf  []byte
		data any
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "write nil",
			args: args{data: nil},
			want: nil,
		},
		{
			name: "append nil",
			args: args{buf: []byte{1, 2, 3}, data: nil},
			want: []byte{1, 2, 3},
		},
		{
			name: "write fixed size data",
			args: args{data: uint8(4)},
			want: []byte{4},
		},
		{
			name: "append fixed size data",
			args: args{buf: []byte{1, 2, 3}, data: uint8(4)},
			want: []byte{1, 2, 3, 4},
		},
		{
			name: "write []byte",
			args: args{data: []byte{4, 5, 6}},
			want: []byte{4, 5, 6},
		},
		{
			name: "append []byte",
			args: args{buf: []byte{1, 2, 3}, data: []byte{4, 5, 6}},
			want: []byte{1, 2, 3, 4, 5, 6},
		},
		{
			name: "write string",
			args: args{data: "AAA"},
			want: []byte{65, 65, 65},
		},
		{
			name: "append string",
			args: args{buf: []byte{1, 2, 3}, data: "AAA"},
			want: []byte{1, 2, 3, 65, 65, 65},
		},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bytes.NewBuffer(tt.args.buf)
			SafeWriteBuf(b, tt.args.data)
			require.EqualValues(tt.want, b.Bytes())
		})
	}
}

func TestReadWriteShortString(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "empty string",
			args: args{str: ""},
		},
		{
			name: "basic",
			args: args{str: "AAA"},
		},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bytes.NewBuffer(nil)
			WriteShortString(b, tt.args.str)

			s, err := ReadShortString(b)
			require.NoError(err)
			require.EqualValues(tt.args.str, s)
		})
	}

	t.Run("must be truncated on write large (> 64K) string", func(t *testing.T) {
		str := strings.Repeat("A", 0xFFFF+1)

		b := bytes.NewBuffer(nil)
		WriteShortString(b, str)

		result, err := ReadShortString(b)
		require.NoError(err)
		require.EqualValues(strings.Repeat("A", 0xFFFF), result)
	})

	t.Run("must be error read from EOF buffer", func(t *testing.T) {
		b := bytes.NewBuffer(nil)
		_, err := ReadShortString(b)

		require.ErrorIs(err, io.ErrUnexpectedEOF)
	})

	t.Run("must be error if not enough chars to read", func(t *testing.T) {
		b := bytes.NewBuffer([]byte{0, 3, 65, 65})
		_, err := ReadShortString(b)

		require.ErrorIs(err, io.ErrUnexpectedEOF)
		require.ErrorContains(err, "expected 3 bytes, but only 2")
	})
}

func TestReadXXX(t *testing.T) {
	type s struct {
		int8
		byte
		bool
		int16
		uint16
		int32
		uint32
		int64
		uint64
		float32
		float64
		string
	}
	s1 := s{
		int8:    -1,
		byte:    1,
		bool:    true,
		int16:   -2222,
		uint16:  3333,
		int32:   -444444,
		uint32:  555555,
		int64:   -66666666666,
		uint64:  77777777777,
		float32: -8.888e8,
		float64: 9.9999e99,
		string:  "test 🧪 test",
	}

	buf := new(bytes.Buffer)
	SafeWriteBuf(buf, s1.int8)
	SafeWriteBuf(buf, s1.byte)
	SafeWriteBuf(buf, s1.bool)
	SafeWriteBuf(buf, s1.int16)
	SafeWriteBuf(buf, s1.uint16)
	SafeWriteBuf(buf, s1.int32)
	SafeWriteBuf(buf, s1.uint32)
	SafeWriteBuf(buf, s1.int64)
	SafeWriteBuf(buf, s1.uint64)
	SafeWriteBuf(buf, s1.float32)
	SafeWriteBuf(buf, s1.float64)
	WriteShortString(buf, s1.string)

	data := buf.Bytes()

	t.Run("Read×××", func(t *testing.T) {
		require := require.New(t)

		s2 := s{}
		buf := bytes.NewBuffer(data)

		var e error

		s2.int8, e = ReadInt8(buf)
		require.NoError(e)
		s2.byte, e = ReadByte(buf)
		require.NoError(e)
		s2.bool, e = ReadBool(buf)
		require.NoError(e)
		s2.int16, e = ReadInt16(buf)
		require.NoError(e)
		s2.uint16, e = ReadUInt16(buf)
		require.NoError(e)
		s2.int32, e = ReadInt32(buf)
		require.NoError(e)
		s2.uint32, e = ReadUInt32(buf)
		require.NoError(e)
		s2.int64, e = ReadInt64(buf)
		require.NoError(e)
		s2.uint64, e = ReadUInt64(buf)
		require.NoError(e)
		s2.float32, e = ReadFloat32(buf)
		require.NoError(e)
		s2.float64, e = ReadFloat64(buf)
		require.NoError(e)
		s2.string, e = ReadShortString(buf)
		require.NoError(e)

		require.EqualValues(s1, s2)
	})
}

func TestReadXXXerrors(t *testing.T) {
	var e error
	require := require.New(t)

	b := bytes.NewBuffer([]byte{0})
	_ = b.Next(1)

	_, e = ReadInt8(b)
	require.ErrorIs(e, io.ErrUnexpectedEOF)

	_, e = ReadByte(b)
	require.ErrorIs(e, io.ErrUnexpectedEOF)

	_, e = ReadBool(b)
	require.ErrorIs(e, io.ErrUnexpectedEOF)

	_, e = ReadInt16(b)
	require.ErrorIs(e, io.ErrUnexpectedEOF)

	_, e = ReadUInt16(b)
	require.ErrorIs(e, io.ErrUnexpectedEOF)

	_, e = ReadInt32(b)
	require.ErrorIs(e, io.ErrUnexpectedEOF)

	_, e = ReadUInt32(b)
	require.ErrorIs(e, io.ErrUnexpectedEOF)

	_, e = ReadInt64(b)
	require.ErrorIs(e, io.ErrUnexpectedEOF)

	_, e = ReadUInt64(b)
	require.ErrorIs(e, io.ErrUnexpectedEOF)

	_, e = ReadFloat32(b)
	require.ErrorIs(e, io.ErrUnexpectedEOF)

	_, e = ReadFloat64(b)
	require.ErrorIs(e, io.ErrUnexpectedEOF)
}

func TestCopyBytes(t *testing.T) {
	type args struct {
		src []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "basic test",
			args: args{src: []byte{1, 2, 3}},
			want: []byte{1, 2, 3},
		},
		{
			name: "must be ok to copy from nil",
			args: args{src: nil},
			want: []byte{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyBytes(tt.args.src); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrefixBytes(t *testing.T) {
	type args struct {
		bytes  []byte
		prefix []interface{}
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "add uint16 to two bytes",
			args: args{
				bytes:  []byte{0x01, 0x02},
				prefix: []interface{}{uint16(1)},
			},
			want: []byte{0x00, 0x01, 0x01, 0x02},
		},
		{
			name: "add uint16 and uint64",
			args: args{
				bytes:  []byte{0x01, 0x02},
				prefix: []interface{}{uint16(0x0107), uint64(0xA7010203)},
			},
			want: []byte{0x01, 0x07, 0x00, 0x00, 0x00, 0x00, 0xA7, 0x01, 0x02, 0x03, 0x01, 0x02},
		},
		{
			name: "add uint16 and uint64 to nil",
			args: args{
				bytes:  nil,
				prefix: []interface{}{uint16(0x0107), uint64(0xA7010203)},
			},
			want: []byte{0x01, 0x07, 0x00, 0x00, 0x00, 0x00, 0xA7, 0x01, 0x02, 0x03},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PrefixBytes(tt.args.bytes, tt.args.prefix...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("prefixBytes() = %v, want %v", got, tt.want)
			}
		})
	}

	require.New(t).Panics(func() {
		bytes := []byte{0x01, 0x02}
		const value = 55 // unknown type size!
		_ = PrefixBytes(bytes, value)
	}, "must panic if expand bytes slice by unknown/variable size values")
}

type testInt uint16

func TestToBytes(t *testing.T) {
	type args struct {
		value []interface{}
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "fixed width",
			args: args{value: []interface{}{uint16(20)}},
			want: []byte{0, 20},
		},
		{
			name: "fixed width custom type",
			args: args{value: []interface{}{testInt(1973)}},
			want: []byte{0x07, 0xb5},
		},
		{
			name: "[]byte",
			args: args{value: []interface{}{[]byte{1, 2, 3}}},
			want: []byte{1, 2, 3},
		},
		{
			name: "string",
			args: args{value: []interface{}{"AAA"}},
			want: []byte{65, 65, 65},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToBytes(tt.args.value...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFullBytes(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nil case",
			args: args{b: nil},
			want: true,
		},
		{
			name: "null len case",
			args: args{b: []byte{}},
			want: true,
		},
		{
			name: "full byte test",
			args: args{b: []byte{0xFF}},
			want: true,
		},
		{
			name: "full word test",
			args: args{b: []byte{0xFF, 0xFF}},
			want: true,
		},
		{
			name: "full long bytes test",
			args: args{b: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}},
			want: true,
		},
		{
			name: "negative test",
			args: args{b: []byte("bytes")},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FullBytes(tt.args.b); got != tt.want {
				t.Errorf("fullBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIncBytes(t *testing.T) {
	type args struct {
		cc []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "nil test",
			args: args{cc: nil},
			want: nil,
		},
		{
			name: "null len test",
			args: args{cc: []byte{}},
			want: nil,
		},
		{
			name: "full byte test",
			args: args{cc: []byte{0xFF}},
			want: nil,
		},
		{
			name: "full word test",
			args: args{cc: []byte{0xFF, 0xFF}},
			want: nil,
		},
		{
			name: "basic test",
			args: args{cc: []byte{0x01, 0x02}},
			want: []byte{0x01, 0x03},
		},
		{
			name: "full-end test",
			args: args{cc: []byte{0x01, 0xFF}},
			want: []byte{0x02, 0x00},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotFinishCCols := IncBytes(tt.args.cc); !reflect.DeepEqual(gotFinishCCols, tt.want) {
				t.Errorf("rangeCCols() = %v, want %v", gotFinishCCols, tt.want)
			}
		})
	}
}
