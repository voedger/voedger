/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qnames

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func TestQNames(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	sp := istorageimpl.Provide(mem.Provide(testingu.MockTime))
	storage, err := sp.AppStorage(appName)
	require.NoError(err)

	versions := vers.New()
	if err := versions.Prepare(storage); err != nil {
		panic(err)
	}

	defName := appdef.NewQName("test", "doc")

	names := New()
	if err := names.Prepare(storage, versions,
		func() appdef.IAppDef {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			ws := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
			ws.AddCDoc(defName)
			appDef, err := adb.Build()
			require.NoError(err)
			return appDef
		}()); err != nil {
		panic(err)
	}

	t.Run("basic QNames methods", func(t *testing.T) {

		check := func(names *QNames, name appdef.QName) istructs.QNameID {
			id, err := names.ID(name)
			require.NoError(err)
			require.NotEqual(istructs.NullQNameID, id)

			n, err := names.QName(id)
			require.NoError(err)
			require.Equal(name, n)

			return id
		}

		sID := check(names, defName)

		t.Run("should be ok to load early stored names", func(t *testing.T) {
			versions1 := vers.New()
			if err := versions1.Prepare(storage); err != nil {
				panic(err)
			}

			names1 := New()
			if err := names1.Prepare(storage, versions, nil); err != nil {
				panic(err)
			}

			require.Equal(sID, check(names1, defName))
		})

		t.Run("should be ok to redeclare names", func(t *testing.T) {
			versions2 := vers.New()
			if err := versions2.Prepare(storage); err != nil {
				panic(err)
			}

			names2 := New()
			if err := names2.Prepare(storage, versions,
				func() appdef.IAppDef {
					adb := builder.New()
					adb.AddPackage("test", "test.com/test")
					ws := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
					ws.AddCDoc(defName)
					appDef, err := adb.Build()
					require.NoError(err)
					return appDef
				}()); err != nil {
				panic(err)
			}

			require.Equal(sID, check(names2, defName))
		})
	})

	t.Run("should be error if unknown name", func(t *testing.T) {
		id, err := names.ID(appdef.NewQName("test", "unknown"))
		require.Equal(istructs.NullQNameID, id)
		require.ErrorIs(err, ErrNameNotFound)
	})

	t.Run("should be error if unknown id", func(t *testing.T) {
		n, err := names.QName(istructs.QNameID(MaxAvailableQNameID))
		require.Equal(appdef.NullQName, n)
		require.ErrorIs(err, ErrIDNotFound)
	})
}

func TestQNamesPrepareErrors(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	t.Run("should be error if unknown system view version", func(t *testing.T) {
		sp := istorageimpl.Provide(mem.Provide(testingu.MockTime))
		storage, _ := sp.AppStorage(appName)

		versions := vers.New()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		versions.Put(vers.SysQNamesVersion, latestVersion+1)

		names := New()
		err := names.Prepare(storage, versions, nil)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})

	t.Run("should be error if invalid QName loaded from system view ", func(t *testing.T) {
		sp := istorageimpl.Provide(mem.Provide(testingu.MockTime))
		storage, _ := sp.AppStorage(appName)

		versions := vers.New()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		versions.Put(vers.SysQNamesVersion, latestVersion)
		const badName = "-test.error.qname-"
		storage.Put(utils.ToBytes(consts.SysView_QNames, ver01), []byte(badName), utils.ToBytes(istructs.QNameID(512)))

		names := New()
		err := names.Prepare(storage, versions, nil)
		require.ErrorIs(err, appdef.ErrConvertError)
		require.ErrorContains(err, badName)
	})

	t.Run("should be ok if deleted QName loaded from system view ", func(t *testing.T) {
		sp := istorageimpl.Provide(mem.Provide(testingu.MockTime))
		storage, _ := sp.AppStorage(appName)

		versions := vers.New()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		versions.Put(vers.SysQNamesVersion, latestVersion)
		storage.Put(utils.ToBytes(consts.SysView_QNames, ver01), []byte("test.deleted"), utils.ToBytes(istructs.NullQNameID))

		names := New()
		err := names.Prepare(storage, versions, nil)
		require.NoError(err)
	})

	t.Run("should be error if invalid (small) istructs.QNameID loaded from system view ", func(t *testing.T) {
		sp := istorageimpl.Provide(mem.Provide(testingu.MockTime))
		storage, _ := sp.AppStorage(appName)

		versions := vers.New()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		versions.Put(vers.SysQNamesVersion, latestVersion)
		storage.Put(utils.ToBytes(consts.SysView_QNames, ver01), []byte(istructs.QNameForError.String()), utils.ToBytes(istructs.QNameIDForError))

		names := New()
		err := names.Prepare(storage, versions, nil)
		require.ErrorIs(err, ErrWrongQNameID)
		require.ErrorContains(err, fmt.Sprintf("unexpected ID (%v)", istructs.QNameIDForError))
	})

	t.Run("should be error if too many QNames", func(t *testing.T) {
		sp := istorageimpl.Provide(mem.Provide(testingu.MockTime))
		storage, _ := sp.AppStorage(appName)

		versions := vers.New()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		names := New()
		err := names.Prepare(storage, versions,
			func() appdef.IAppDef {
				adb := builder.New()
				adb.AddPackage("test", "test.com/test")
				wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
				for i := range MaxAvailableQNameID + 1 {
					wsb.AddObject(appdef.NewQName("test", fmt.Sprintf("name_%d", i)))
				}
				appDef, err := adb.Build()
				require.NoError(err)
				return appDef
			}())
		require.ErrorIs(err, ErrQNameIDsExceeds)
	})

	t.Run("should be error if write to storage failed", func(t *testing.T) {
		if testing.Short() {
			t.Skip()
		}
		qName := appdef.NewQName("test", "test")
		writeError := errors.New("storage write error")

		t.Run("should be error if write some name failed", func(t *testing.T) {
			storage := teststore.NewStorage(appName)

			versions := vers.New()
			if err := versions.Prepare(storage); err != nil {
				panic(err)
			}

			storage.SchedulePutError(writeError, utils.ToBytes(consts.SysView_QNames, ver01), []byte(qName.String()))

			names := New()
			err := names.Prepare(storage, versions,
				func() appdef.IAppDef {
					adb := builder.New()
					adb.AddPackage("test", "test.com/test")
					wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
					wsb.AddObject(qName)
					appDef, err := adb.Build()
					require.NoError(err)
					return appDef
				}())
			require.ErrorIs(err, writeError)
		})

		t.Run("should be error if write system view version failed", func(t *testing.T) {
			storage := teststore.NewStorage(appName)

			versions := vers.New()
			if err := versions.Prepare(storage); err != nil {
				panic(err)
			}

			storage.SchedulePutError(writeError, utils.ToBytes(consts.SysView_Versions), utils.ToBytes(vers.SysQNamesVersion))

			names := New()
			err := names.Prepare(storage, versions,
				func() appdef.IAppDef {
					adb := builder.New()
					adb.AddPackage("test", "test.com/test")
					wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
					wsb.AddObject(qName)
					appDef, err := adb.Build()
					require.NoError(err)
					return appDef
				}())
			require.ErrorIs(err, writeError)
		})
	})
}
