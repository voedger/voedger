/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package set

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmpty(t *testing.T) {
	require := require.New(t)
	require.Zero(Empty[byte]().Len())
	require.Empty(Empty[byte]().AsArray())
	require.EqualValues(`[]`, Empty[byte]().String())
	v, ok := Empty[byte]().First()
	require.False(ok)
	require.Zero(v)
}

func TestFrom(t *testing.T) {
	require := require.New(t)

	tests := []struct {
		name string
		set  Set[uint8]
		want string
	}{
		{"empty", From[uint8](), "[]"},
		{"1 63", From(uint8(1), 63), "[1 63]"},
		{"1 63 64 127", From(uint8(1), 63, 64, 127), "[1 63 64 127]"},
		{"1 63 64 127 128 255", From(uint8(1), 63, 64, 127, 128, 255), "[1 63 64 127 128 255]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, tt.set.String(), "SetFrom(%v).String() = %v, want %v", tt.set, tt.set.String(), tt.want)
		})
	}
}

func TestSet_All(t *testing.T) {
	require := require.New(t)

	set := From[uint8](0, 1, 2, 3, 126, 127, 128, 129, 253, 254, 255)

	var sum int
	for v := range set.All {
		sum += int(v)
	}
	require.EqualValues(1278, sum)

	t.Run("should be breakable", func(t *testing.T) {
		var sum int
		for v := range set.All {
			sum += int(v)
			if v > 128 {
				break
			}
		}
		require.EqualValues(0+1+2+3+126+127+128+129, sum)
	})
}

func TestSet_AsArray(t *testing.T) {
	require := require.New(t)

	tests := []struct {
		name string
		set  Set[uint8]
		want []uint8
	}{
		{"empty", Empty[uint8](), nil},
		{"0 63", From(uint8(0), 63), []uint8{0, 63}},
		{"0 63 64 127", From(uint8(0), 63, 64, 127), []uint8{0, 63, 64, 127}},
		{"0 63 64 127 128 255", From(uint8(0), 63, 64, 127, 128, 255), []uint8{0, 63, 64, 127, 128, 255}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.set.AsArray()
			require.EqualValues(tt.want, got, "SetFrom(%v).AsArray() = %v, want %v", tt.set, got, tt.want)
		})
	}
}

func TestSet_AsBytes(t *testing.T) {
	require := require.New(t)

	tests := []struct {
		name string
		set  Set[uint8]
		want []byte
	}{
		{"empty", Empty[uint8](), []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{"0", From[uint8](0), []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
		{"0 1 127 128", From[uint8](0, 1, 127, 128), []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0b00000001, 0b10000000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0b00000011}},
		{"0 1 127 128 191 192", From[uint8](0, 1, 127, 128, 191, 192), []byte{0, 0, 0, 0, 0, 0, 0, 0b000000001, 0b10000000, 0, 0, 0, 0, 0, 0, 0b00000001, 0b10000000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0b00000011}},
		{"0 1 127 128 191 192 253 254 255", From[uint8](0, 1, 127, 128, 191, 192, 253, 254, 255), []byte{0b11100000, 0, 0, 0, 0, 0, 0, 0b000000001, 0b10000000, 0, 0, 0, 0, 0, 0, 0b00000001, 0b10000000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0b00000011}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.set.AsBytes()
			require.EqualValues(tt.want, got, "SetFrom(%v).AsBytes() = %v, want %v", tt.set, got, tt.want)
		})
	}
}

func TestSet_Backward(t *testing.T) {
	require := require.New(t)

	set := From[uint8](0, 1, 2, 3, 126, 127, 128, 129, 253, 254, 255)
	set.SetReadOnly()
	result := make([]int, 0, set.Len())
	for v := range set.Backward {
		result = append(result, int(v))
	}
	require.EqualValues([]int{255, 254, 253, 129, 128, 127, 126, 3, 2, 1, 0}, result)

	t.Run("should be breakable", func(t *testing.T) {
		set := From[uint8](0, 1, 2, 3, 129, 253, 254, 255)
		result := make([]int, 0, set.Len())
		for v := range set.Backward {
			result = append(result, int(v))
			if v < 128 {
				break
			}
		}
		require.EqualValues([]int{255, 254, 253, 129, 3}, result)
	})
}

func TestSet_Chunk(t *testing.T) {
	require := require.New(t)

	type byteSet = Set[byte]
	set := From[uint8](0, 1, 2, 3, 126, 127, 128, 129, 253, 254, 255)
	set.SetReadOnly()
	i := 0
	for v := range set.Chunk(3) {
		switch i {
		case 0:
			require.EqualValues(From(byte(0), 1, 2), v)
		case 1:
			require.EqualValues(From(byte(3), 126, 127), v)
		case 2:
			require.EqualValues(From(byte(128), 129, 253), v)
		case 3:
			require.EqualValues(From(byte(254), 255), v)
		default:
			require.Fail("unexpected chunk", "chunk %d: %v", i, v)
		}
		i++
	}

	t.Run("should be breakable", func(t *testing.T) {
		i := 0
		for v := range set.Chunk(3) {
			i++
			require.EqualValues(From(byte(0), 1, 2), v)
			break
		}
		require.Equal(1, i)
	})

	for range Empty[byte]().Chunk(1) {
		require.Fail("should be no visits for empty set")
	}

	t.Run("should be panics if n < 1", func(t *testing.T) {
		require.Panics(func() {
			for range set.Chunk(0) {
			}
		})
	})
}

func TestSet_Clear(t *testing.T) {
	require := require.New(t)

	set := From[uint8](0, 1, 2, 3, 126, 127, 128, 129, 253, 254, 255)

	// clear odd
	set.Clear(1, 3, 127, 129, 253, 255)
	require.Equal("[0 2 126 128 254]", set.String())

	// clear even
	set.Clear(0, 2, 126, 128, 254)
	require.Equal("[]", set.String())
}

func TestSet_ClearAll(t *testing.T) {
	require := require.New(t)

	set := From[uint8](0, 1, 2, 3, 63, 64, 65, 66, 67, 126, 127, 128, 129, 191, 192, 193, 252, 253, 254, 255)
	set.ClearAll()
	require.Equal("[]", set.String())
	require.Zero(set.Len())
	require.Empty(set.AsArray())
}

func TestSet_Clone(t *testing.T) {
	require := require.New(t)

	tests := []struct {
		name string
		set  Set[uint8]
	}{
		{"empty", Set[uint8]{}},
		{"one", From(uint8(128))},
		{"two", From[uint8](128, 247)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clone := tt.set.Clone()
			require.Equal(tt.set.String(), clone.String())
			require.Equal(tt.set.Len(), clone.Len())
			require.Equal(tt.set.AsArray(), clone.AsArray())

			clone.Set(1)

			require.NotEqual(tt.set.String(), clone.String())
			require.Equal(tt.set.Len()+1, clone.Len())
			require.NotEqual(tt.set.AsArray(), clone.AsArray())
		})
	}
}

func TestSet_Contains(t *testing.T) {
	tests := []struct {
		name string
		set  Set[byte]
		v    byte
		want bool
	}{
		{"empty", Set[byte]{}, 5, false},
		{"one", From[byte](155), 155, true},
		{"two", From[byte](128, 194), 194, true},
		{"negative", From[byte](128, 194), 250, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.Contains(tt.v); got != tt.want {
				t.Errorf("Set(%v).Contains(%v) = %v, want %v", tt.set, tt.v, got, tt.want)
			}
		})
	}
}

func TestSet_ContainsAll(t *testing.T) {
	tests := []struct {
		name   string
		set    Set[byte]
		values []byte
		want   bool
	}{
		{"nil in empty", Set[byte]{}, nil, true},
		{"empty in empty", Set[byte]{}, []byte{}, true},
		{"100 in empty", Set[byte]{}, []byte{100}, false},
		{"100 in [100]", From[byte](100), []byte{100}, true},
		{"100 & 101 in [100]", From[byte](100), []byte{100, 101}, false},
		{"100 & 101 in [100, 101]", From[byte](100, 101), []byte{100, 101}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.ContainsAll(tt.values...); got != tt.want {
				t.Errorf("Set(%v).ContainsAll(%v) = %v, want %v", tt.set, tt.values, got, tt.want)
			}
		})
	}
}

func TestSet_ContainsAny(t *testing.T) {
	tests := []struct {
		name   string
		set    Set[byte]
		values []byte
		want   bool
	}{
		{"nil in empty", Set[byte]{}, nil, true},
		{"empty in empty", Set[byte]{}, []byte{}, true},
		{"100 in empty", Set[byte]{}, []byte{100}, false},
		{"100 in [100]", From[byte](100), []byte{100}, true},
		{"100 & 101 in [100]", From[byte](100), []byte{100, 101}, true},
		{"100 & 101 in [50, 150]", From[byte](50, 150), []byte{100, 101}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.ContainsAny(tt.values...); got != tt.want {
				t.Errorf("Set(%v).ContainsAny(%v) = %v, want %v", tt.set, tt.values, got, tt.want)
			}
		})
	}
}

func TestSet_First(t *testing.T) {
	tests := []struct {
		name      string
		set       Set[byte]
		wantValue byte
		wantOk    bool
	}{
		{"empty", Set[byte]{}, 0, false},
		{"one", From(byte(100)), 100, true},
		{"two", From(byte(100), 200), 100, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotOk := tt.set.First()
			if !reflect.DeepEqual(gotValue, tt.wantValue) {
				t.Errorf("Set(%v).First() got value = %v, want %v", tt.set, gotValue, tt.wantValue)
			}
			if gotOk != tt.wantOk {
				t.Errorf("Set(%v).First() got ok = %v, want %v", tt.set, gotOk, tt.wantOk)
			}
		})
	}
}

func TestSet_Len(t *testing.T) {
	tests := []struct {
		name string
		set  Set[byte]
		want int
	}{
		{"empty", Set[byte]{}, 0},
		{"one", From(byte(100)), 1},
		{"two", From(byte(100), 200), 2},
		{"ten", From(byte(0), 1, 127, 128, 129, 191, 192, 193, 254, 255), 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.Len(); got != tt.want {
				t.Errorf("Set(%v).Len() = %v, want %v", tt.set, got, tt.want)
			}
		})
	}
}

func TestSet_SetRange(t *testing.T) {
	type args struct {
		start byte
		end   byte
	}
	tests := []struct {
		name string
		set  Set[byte]
		args args
		want string
	}{
		{"empty", Set[byte]{}, args{127, 127}, "[]"},
		{"one", Set[byte]{}, args{127, 127 + 1}, "[127]"},
		{"two", Set[byte]{}, args{127, 127 + 2}, "[127 128]"},
		{"two + range", From(byte(32), 64), args{127, 127 + 2}, "[32 64 127 128]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.set.SetRange(tt.args.start, tt.args.end)
			if got := tt.set.String(); got != tt.want {
				t.Errorf("Set.SetRange(%v, %v).String() = %v, want %v", tt.args.start, tt.args.end, got, tt.want)
			}
		})
	}
}

type TestByte byte

func (b TestByte) TrimString() string {
	return fmt.Sprintf("%#02x", b)
}

func TestSet_String(t *testing.T) {
	tests := []struct {
		name string
		set  any
		want string
	}{
		{"empty", Set[byte]{}, "[]"},
		{"one", From[byte](100), "[100]"},
		{"many", From[byte](0, 3, 63, 65, 127, 129, 191, 193, 253, 255), "[0 3 63 65 127 129 191 193 253 255]"},
		{"with TrimString", From[TestByte](0, 1, 3, 63, 65, 127, 129, 191, 193, 253, 255), "[0x00 0x01 0x03 0x3f 0x41 0x7f 0x81 0xbf 0xc1 0xfd 0xff]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fmt.Sprint(tt.set); got != tt.want {
				t.Errorf("Set(%v).String() = %v, want %v", tt.set, got, tt.want)
			}
		})
	}
}

func Test_SetReadOnly(t *testing.T) {
	require := require.New(t)

	s := From[byte](0, 1, 2, 3)
	s.SetReadOnly()

	t.Run("should panics if", func(t *testing.T) {
		require.Panics(func() { s.Clear(1) }, "Clear")
		require.Panics(func() { s.ClearAll() }, "ClearAll")
		require.Panics(func() { s.Set(1) }, "Set")
		require.Panics(func() { s.SetRange(1, 3) }, "SetRange")
	})
}
