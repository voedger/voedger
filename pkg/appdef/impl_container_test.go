/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_type_AddContainer(t *testing.T) {
	require := require.New(t)

	adb := New()
	adb.AddPackage("test", "test.com/test")

	rootName := NewQName("test", "root")
	root := adb.AddObject(rootName)
	require.NotNil(root)

	childName := NewQName("test", "child")
	_ = adb.AddObject(childName)

	t.Run("must be ok to add container", func(t *testing.T) {
		root.AddContainer("c1", childName, 1, Occurs_Unbounded)

		app, err := adb.Build()
		require.NoError(err)
		require.NotNil(app)

		root := app.Object(rootName)
		require.NotNil(root)

		require.EqualValues(1, root.ContainerCount())

		c1 := root.Container("c1")
		require.NotNil(c1)
		require.EqualValues(childName, c1.Type().QName())
	})

	t.Run("chain notation is ok to add containers", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		_ = adb.AddObject(childName)
		_ = adb.AddObject(rootName).
			AddContainer("c1", childName, 1, Occurs_Unbounded).
			AddContainer("c2", childName, 1, Occurs_Unbounded).
			AddContainer("c3", childName, 1, Occurs_Unbounded)

		app, err := adb.Build()
		require.NoError(err)

		obj := app.Object(rootName)
		require.NotNil(obj)
		require.EqualValues(3, obj.ContainerCount())
		require.NotNil(obj.Container("c1"))
		require.NotNil(obj.Container("c2"))
		require.NotNil(obj.Container("c3"))
		require.Nil(obj.Container("unknown"))
	})

	t.Run("must be panic if empty container name", func(t *testing.T) {
		require.Panics(func() { root.AddContainer("", childName, 1, Occurs_Unbounded) },
			require.Is(ErrMissedError))
	})

	t.Run("must be panic if invalid container name", func(t *testing.T) {
		require.Panics(func() { root.AddContainer("naked_🔫", childName, 1, Occurs_Unbounded) },
			require.Is(ErrInvalidError))
	})

	t.Run("must be panic if container name dupe", func(t *testing.T) {
		require.Panics(func() { root.AddContainer("c1", childName, 1, Occurs_Unbounded) },
			require.Is(ErrAlreadyExistsError),
			require.Has("c1"))
	})

	t.Run("must be panic if container type name missed", func(t *testing.T) {
		require.Panics(func() { root.AddContainer("c2", NullQName, 1, Occurs_Unbounded) },
			require.Is(ErrMissedError),
			require.Has("c2"))
	})

	t.Run("must be panic if invalid occurrences", func(t *testing.T) {
		require.Panics(func() { root.AddContainer("c2", childName, 1, 0) },
			require.Is(ErrOutOfBoundsError))
		require.Panics(func() { root.AddContainer("c3", childName, 2, 1) },
			require.Is(ErrOutOfBoundsError))
	})

	t.Run("must be panic if container type is incompatible", func(t *testing.T) {
		docName := NewQName("test", "doc")
		_ = adb.AddCDoc(docName)
		require.Panics(func() { root.AddContainer("c2", docName, 1, 1) },
			require.Is(ErrInvalidError),
			require.Has(docName.String()))
	})

	t.Run("must be panic if too many containers", func(t *testing.T) {
		el := New().AddObject(childName)
		for i := 0; i < MaxTypeContainerCount; i++ {
			el.AddContainer(fmt.Sprintf("c_%#x", i), childName, 0, Occurs_Unbounded)
		}
		require.Panics(func() { el.AddContainer("errorContainer", childName, 0, Occurs_Unbounded) },
			require.Is(ErrTooManyError))
	})
}

func TestValidateContainer(t *testing.T) {
	require := require.New(t)

	app := New()
	app.AddPackage("test", "test.com/test")
	doc := app.AddCDoc(NewQName("test", "doc"))
	doc.AddContainer("rec", NewQName("test", "rec"), 0, Occurs_Unbounded)

	t.Run("must be error if container type not found", func(t *testing.T) {
		_, err := app.Build()
		require.Error(err, require.Is(ErrNotFoundError), require.Has("test.rec"))
	})

	rec := app.AddCRecord(NewQName("test", "rec"))
	_, err := app.Build()
	require.NoError(err)

	t.Run("must be ok container recurse", func(t *testing.T) {
		rec.AddContainer("rec", NewQName("test", "rec"), 0, Occurs_Unbounded)
		_, err := app.Build()
		require.NoError(err)
	})

	t.Run("must be ok container sub recurse", func(t *testing.T) {
		rec.AddContainer("rec1", NewQName("test", "rec1"), 0, Occurs_Unbounded)
		rec1 := app.AddCRecord(NewQName("test", "rec1"))
		rec1.AddContainer("rec", NewQName("test", "rec"), 0, Occurs_Unbounded)
		_, err := app.Build()
		require.NoError(err)
	})

	t.Run("must be error if container kind is incompatible", func(t *testing.T) {
		doc.AddContainer("obj", NewQName("test", "obj"), 0, 1)
		_ = app.AddObject(NewQName("test", "obj"))
		_, err := app.Build()
		require.Error(err, require.Is(ErrInvalidError), require.Has("test.obj"))
	})
}

func TestOccurs_String(t *testing.T) {
	tests := []struct {
		name string
		o    Occurs
		want string
	}{
		{
			name: "0 —> `0`",
			o:    0,
			want: `0`,
		},
		{
			name: "1 —> `1`",
			o:    1,
			want: `1`,
		},
		{
			name: "∞ —> `unbounded`",
			o:    Occurs_Unbounded,
			want: `unbounded`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.String(); got != tt.want {
				t.Errorf("Occurs.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOccurs_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		o       Occurs
		want    string
		wantErr bool
	}{
		{
			name:    "0 —> `0`",
			o:       0,
			want:    `0`,
			wantErr: false,
		},
		{
			name:    "1 —> `1`",
			o:       1,
			want:    `1`,
			wantErr: false,
		},
		{
			name:    "∞ —> `unbounded`",
			o:       Occurs_Unbounded,
			want:    `"unbounded"`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.o.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("Occurs.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(got) != tt.want {
				t.Errorf("Occurs.MarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOccurs_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    Occurs
		wantErr bool
	}{
		{
			name:    "0 —> 0",
			data:    `0`,
			want:    0,
			wantErr: false,
		},
		{
			name:    "1 —> 1",
			data:    `1`,
			want:    1,
			wantErr: false,
		},
		{
			name:    `"unbounded" —> ∞`,
			data:    `"unbounded"`,
			want:    Occurs_Unbounded,
			wantErr: false,
		},
		{
			name:    `"3" —> error`,
			data:    `"3"`,
			want:    0,
			wantErr: true,
		},
		{
			name:    `65536 —> error`,
			data:    `65536`,
			want:    0,
			wantErr: true,
		},
		{
			name:    `-1 —> error`,
			data:    `-1`,
			want:    0,
			wantErr: true,
		},
		{
			name:    `"abc" —> error`,
			data:    `"abc"`,
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var o Occurs
			err := o.UnmarshalJSON([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("Occurs.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if o != tt.want {
					t.Errorf("o.UnmarshalJSON() result = %v, want %v", o, tt.want)
				}
			}
		})
	}
}
