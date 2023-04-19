package utils

import (
	"bytes"
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
			name: "vulgaris",
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

		require.ErrorContains(err, "length")
	})

	t.Run("must be error if not enouth chars to read", func(t *testing.T) {
		b := bytes.NewBuffer([]byte{0, 3, 65, 65})
		_, err := ReadShortString(b)

		require.ErrorContains(err, "expected 3 bytes, but only 2")
	})
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
			name: "vulgaris test",
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

func TestSuccBytes(t *testing.T) {
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
			name: "vulgaris test",
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
			if gotFinishCCols := SuccBytes(tt.args.cc); !reflect.DeepEqual(gotFinishCCols, tt.want) {
				t.Errorf("rangeCCols() = %v, want %v", gotFinishCCols, tt.want)
			}
		})
	}
}
