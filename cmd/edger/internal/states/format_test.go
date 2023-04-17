/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package states

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAttributeKind_String(t *testing.T) {
	tests := []struct {
		name string
		k    AttributeKind
		want string
	}{
		{name: `0 —> "DockerStackAttribute"`,
			k:    DockerStackAttribute,
			want: `DockerStackAttribute`,
		},
		{name: `out of bounds test`,
			k:    AttributeKindCount + 1,
			want: fmt.Sprintf("AttributeKind(%d)", AttributeKindCount+1),
			//want: "AttributeKind(" + strconv.FormatInt(int64(StateAttributesCount)+1, 10) + ")",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.k.String()
			if got != tt.want {
				t.Errorf("AttributeKind.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttributeKind_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		k    AttributeKind
		want string
	}{
		{name: `0 —> "DockerStackAttribute"`,
			k:    DockerStackAttribute,
			want: `"DockerStackAttribute"`,
		},
		{name: `out of bounds`,
			k:    AttributeKindCount,
			want: strconv.FormatUint(uint64(AttributeKindCount), 10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalJson()
			if err != nil {
				t.Errorf("AttributeKind.MarshalJSON() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("AttributeKind.MarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttributeKind_Unmarshal(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    AttributeKind
		wantErr error
	}{
		{name: `-1 —> UndefinedAttribute`,
			json: `-1`,
			want: UndefinedAttribute,
		},
		{name: `"DockerStackAttribute" —> DockerStackAttribute`,
			json: `"DockerStackAttribute"`,
			want: DockerStackAttribute,
		},
		{name: `0 —> DockerStackAttribute`,
			json: `0`,
			want: DockerStackAttribute,
		},
		{name: fmt.Sprintf(`%d —> %d`, AttributeKindCount, AttributeKindCount),
			json: strconv.FormatInt(int64(AttributeKindCount), 10),
			want: AttributeKindCount,
		},
		{name: `127 —> 127`,
			json: `127`,
			want: AttributeKind(127),
		},
		{name: `out of bounds [-128…127]`,
			json:    `128`,
			wantErr: strconv.ErrRange,
		},
		{name: `"UnknownValue" —> sytax error`,
			json:    `"UnknownValue"`,
			wantErr: strconv.ErrSyntax,
		},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := UndefinedAttribute
			err := a.UnmarshalJSON([]byte(tt.json))
			if tt.wantErr != nil {
				require.ErrorIs(err, tt.wantErr)
			} else {
				require.NoError(err)
				require.Equal(tt.want, a, "AttributeKind.UnmarshalJSON(`%v`) = %v, want %v", tt.json, a, tt.want)
			}
		})
	}
}

func TestActualStatus_String(t *testing.T) {
	tests := []struct {
		name string
		k    ActualStatus
		want string
	}{
		{name: `0 —> "PendingStatus"`,
			k:    PendingStatus,
			want: `PendingStatus`,
		},
		{name: `out of bounds test`,
			k:    ActualStatusCount + 1,
			want: fmt.Sprintf("ActualStatus(%d)", ActualStatusCount+1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.k.String()
			if got != tt.want {
				t.Errorf("AttributeKind.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActualStatus_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		s    ActualStatus
		want string
	}{
		{name: `0 —> "PendingStatus"`,
			s:    PendingStatus,
			want: `"PendingStatus"`,
		},
		{name: `out of bounds`,
			s:    ActualStatusCount,
			want: strconv.FormatUint(uint64(ActualStatusCount), 10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.MarshalJson()
			if err != nil {
				t.Errorf("ActualStatus.MarshalJSON() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("ActualStatus.MarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActualStatus_Unmarshal(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    ActualStatus
		wantErr error
	}{
		{name: `-1 —> UndefinedStatus`,
			json: `-1`,
			want: UndefinedStatus,
		},
		{name: `"PendingStatus" —> PendingStatus`,
			json: `"PendingStatus"`,
			want: PendingStatus,
		},
		{name: `0 —> PendingStatus`,
			json: `0`,
			want: PendingStatus,
		},
		{name: fmt.Sprintf(`%d —> %d`, ActualStatusCount, ActualStatusCount),
			json: strconv.FormatInt(int64(ActualStatusCount), 10),
			want: ActualStatusCount,
		},
		{name: `127 —> 127`,
			json: `127`,
			want: ActualStatus(127),
		},
		{name: `out of bounds [-128…127]`,
			json:    `128`,
			wantErr: strconv.ErrRange,
		},
		{name: `"UnknownValue" —> sytax error`,
			json:    `"UnknownValue"`,
			wantErr: strconv.ErrSyntax,
		},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := UndefinedStatus
			err := a.UnmarshalJSON([]byte(tt.json))
			if tt.wantErr != nil {
				require.ErrorIs(err, tt.wantErr)
			} else {
				require.NoError(err)
				require.Equal(tt.want, a, "ActualStatus.UnmarshalJSON(`%v`) = %v, want %v", tt.json, a, tt.want)
			}
		})
	}
}
