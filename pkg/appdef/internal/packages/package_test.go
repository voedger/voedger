/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package packages_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/packages"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_Packages(t *testing.T) {
	require := require.New(t)

	pkg := packages.MakeWithPackages()

	// should be appdef.IPackages compatible
	var _ appdef.IWithPackages = pkg

	pb := packages.MakePackagesBuilder(&pkg)

	// should be appdef.IPackagesBuilder compatible
	var _ appdef.IPackagesBuilder = &pb

	tests := []struct {
		l, p string
	}{
		{"p1", "test/p1"},
		{"p2", "test/p2"},
	}
	t.Run("should be ok to add packages", func(t *testing.T) {
		for _, tt := range tests {
			pb.AddPackage(tt.l, tt.p)
		}
	})

	t.Run("should be ok to inspect packages", func(t *testing.T) {

		for _, tt := range tests {
			l := appdef.NewQName(tt.l, "name")
			f := appdef.NewFullQName(tt.p, "name")
			require.Equal(f, pkg.FullQName(l))
			require.Equal(l, pkg.LocalQName(f))

			require.Equal(tt.p, pkg.PackageFullPath(tt.l))
			require.Equal(tt.l, pkg.PackageLocalName(tt.p))
		}

		require.Equal(appdef.NullFullQName, pkg.FullQName(appdef.NewQName("test", "unknown")))
		require.Equal(appdef.NullQName, pkg.LocalQName(appdef.NewFullQName("test/unknown", "unknown")))

		t.Run("should be ok to enum packages", func(t *testing.T) {
			p := pkg.Packages()
			require.Len(p, len(tests))
			for _, t := range tests {
				require.Equal(t.p, p[t.l])
			}
		})

		t.Run("should be ok to enum package local names", func(t *testing.T) {
			cnt := 0
			for _, l := range pkg.PackageLocalNames() {
				require.Equal(tests[cnt].l, l)
				cnt++
			}
			require.Equal(len(tests), cnt)
		})
	})

	t.Run("should be ok to use exported functions", func(t *testing.T) {
		packages.AddPackage(&pkg, "local", "full/path")
		require.Equal("local", pkg.PackageLocalName("full/path"))
	})
}

func Test_PackagesPanics(t *testing.T) {
	require := require.New(t)

	pkg := packages.MakeWithPackages()
	pb := packages.MakePackagesBuilder(&pkg)

	t.Run("should be panics", func(t *testing.T) {
		require.Panics(func() {
			pb.AddPackage("naked 🔫", "naked/gun")
		}, require.Is(appdef.ErrInvalidError), require.Has("naked 🔫"))

		pb.AddPackage("dupe", "test/dupe")
		require.Panics(func() {
			pb.AddPackage("dupe", "dupe/test")
		}, require.Is(appdef.ErrAlreadyExistsError), require.Has("dupe"))
		require.Panics(func() {
			pb.AddPackage("other", "test/dupe")
		}, require.Is(appdef.ErrAlreadyExistsError), require.Has("test/dupe"))

		require.Panics(func() {
			pb.AddPackage("empty", "")
		}, require.Is(appdef.ErrMissedError), require.Has("empty"))
	})
}
