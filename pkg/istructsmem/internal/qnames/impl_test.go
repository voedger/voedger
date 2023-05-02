/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qnames

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func TestQNames(t *testing.T) {
	require := require.New(t)

	sp := istorageimpl.Provide(istorage.ProvideMem())
	storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

	versions := vers.New()
	if err := versions.Prepare(storage); err != nil {
		panic(err)
	}

	defName := appdef.NewQName("test", "doc")

	resourceName := appdef.NewQName("test", "resource")
	r := mockResources{}
	r.On("Resources", mock.AnythingOfType("func(appdef.QName)")).
		Run(func(args mock.Arguments) {
			cb := args.Get(0).(func(appdef.QName))
			cb(resourceName)
		})

	names := New()
	if err := names.Prepare(storage, versions,
		func() appdef.IAppDef {
			appDefBuilder := appdef.New()
			appDefBuilder.AddStruct(defName, appdef.DefKind_CDoc)
			appDef, err := appDefBuilder.Build()
			require.NoError(err)
			return appDef
		}(),
		&r); err != nil {
		panic(err)
	}

	t.Run("basic QNames methods", func(t *testing.T) {

		check := func(names *QNames, name appdef.QName) QNameID {
			id, err := names.GetID(name)
			require.NoError(err)
			require.NotEqual(NullQNameID, id)

			n, err := names.GetQName(id)
			require.NoError(err)
			require.Equal(name, n)

			return id
		}

		sID := check(names, defName)
		rID := check(names, resourceName)

		t.Run("must be ok to load early stored names", func(t *testing.T) {
			versions1 := vers.New()
			if err := versions1.Prepare(storage); err != nil {
				panic(err)
			}

			names1 := New()
			if err := names1.Prepare(storage, versions, nil, nil); err != nil {
				panic(err)
			}

			require.Equal(sID, check(names1, defName))
			require.Equal(rID, check(names1, resourceName))
		})

		t.Run("must be ok to redeclare names", func(t *testing.T) {
			versions2 := vers.New()
			if err := versions2.Prepare(storage); err != nil {
				panic(err)
			}

			names2 := New()
			if err := names2.Prepare(storage, versions,
				func() appdef.IAppDef {
					appdefBuilder := appdef.New()
					appdefBuilder.AddStruct(defName, appdef.DefKind_CDoc)
					appDef, err := appdefBuilder.Build()
					require.NoError(err)
					return appDef
				}(),
				nil); err != nil {
				panic(err)
			}

			require.Equal(sID, check(names2, defName))
			require.Equal(rID, check(names2, resourceName))
		})
	})

	t.Run("must be error if unknown name", func(t *testing.T) {
		id, err := names.GetID(appdef.NewQName("test", "unknown"))
		require.Equal(NullQNameID, id)
		require.ErrorIs(err, ErrNameNotFound)
	})

	t.Run("must be error if unknown id", func(t *testing.T) {
		n, err := names.GetQName(QNameID(MaxAvailableQNameID))
		require.Equal(appdef.NullQName, n)
		require.ErrorIs(err, ErrIDNotFound)
	})
}

func TestQNamesPrepareErrors(t *testing.T) {
	require := require.New(t)

	t.Run("must be error if unknown system view version", func(t *testing.T) {
		sp := istorageimpl.Provide(istorage.ProvideMem())
		storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

		versions := vers.New()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		versions.Put(vers.SysQNamesVersion, lastestVersion+1)

		names := New()
		err := names.Prepare(storage, versions, nil, nil)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})

	t.Run("must be error if invalid QName readed from system view ", func(t *testing.T) {
		sp := istorageimpl.Provide(istorage.ProvideMem())
		storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

		versions := vers.New()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		versions.Put(vers.SysQNamesVersion, lastestVersion)
		const badName = "-test.error.qname-"
		storage.Put(utils.ToBytes(consts.SysView_QNames, ver01), []byte(badName), utils.ToBytes(QNameID(512)))

		names := New()
		err := names.Prepare(storage, versions, nil, nil)
		require.ErrorIs(err, appdef.ErrInvalidQNameStringRepresentation)
		require.ErrorContains(err, badName)
	})

	t.Run("must be ok if deleted QName readed from system view ", func(t *testing.T) {
		sp := istorageimpl.Provide(istorage.ProvideMem())
		storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

		versions := vers.New()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		versions.Put(vers.SysQNamesVersion, lastestVersion)
		storage.Put(utils.ToBytes(consts.SysView_QNames, ver01), []byte("test.deleted"), utils.ToBytes(NullQNameID))

		names := New()
		err := names.Prepare(storage, versions, nil, nil)
		require.NoError(err)
	})

	t.Run("must be error if invalid (small) QNameID readed from system view ", func(t *testing.T) {
		sp := istorageimpl.Provide(istorage.ProvideMem())
		storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

		versions := vers.New()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		versions.Put(vers.SysQNamesVersion, lastestVersion)
		storage.Put(utils.ToBytes(consts.SysView_QNames, ver01), []byte(istructs.QNameForError.String()), utils.ToBytes(QNameIDForError))

		names := New()
		err := names.Prepare(storage, versions, nil, nil)
		require.ErrorIs(err, ErrWrongQNameID)
		require.ErrorContains(err, fmt.Sprintf("unexpected ID (%v)", QNameIDForError))
	})

	t.Run("must be error if too many QNames", func(t *testing.T) {
		sp := istorageimpl.Provide(istorage.ProvideMem())
		storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

		versions := vers.New()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		names := New()
		err := names.Prepare(storage, versions,
			func() appdef.IAppDef {
				appDefBuilder := appdef.New()
				for i := 0; i <= MaxAvailableQNameID; i++ {
					appDefBuilder.AddStruct(appdef.NewQName("test", fmt.Sprintf("name_%d", i)), appdef.DefKind_Object)
				}
				appDef, err := appDefBuilder.Build()
				require.NoError(err)
				return appDef
			}(),
			nil)
		require.ErrorIs(err, ErrQNameIDsExceeds)
	})

	t.Run("must be error if write to storage failed", func(t *testing.T) {
		qName := appdef.NewQName("test", "test")
		writeError := errors.New("storage write error")

		t.Run("must be error if write some name failed", func(t *testing.T) {
			storage := teststore.NewStorage()

			versions := vers.New()
			if err := versions.Prepare(storage); err != nil {
				panic(err)
			}

			storage.SchedulePutError(writeError, utils.ToBytes(consts.SysView_QNames, ver01), []byte(qName.String()))

			names := New()
			err := names.Prepare(storage, versions,
				func() appdef.IAppDef {
					appDefBuilder := appdef.New()
					appDefBuilder.AddStruct(qName, appdef.DefKind_Object)
					appDef, err := appDefBuilder.Build()
					require.NoError(err)
					return appDef
				}(),
				nil)
			require.ErrorIs(err, writeError)
		})

		t.Run("must be error if write system view version failed", func(t *testing.T) {
			storage := teststore.NewStorage()

			versions := vers.New()
			if err := versions.Prepare(storage); err != nil {
				panic(err)
			}

			storage.SchedulePutError(writeError, utils.ToBytes(consts.SysView_Versions), utils.ToBytes(vers.SysQNamesVersion))

			names := New()
			err := names.Prepare(storage, versions,
				func() appdef.IAppDef {
					appDefBuilder := appdef.New()
					appDefBuilder.AddStruct(qName, appdef.DefKind_Object)
					appDef, err := appDefBuilder.Build()
					require.NoError(err)
					return appDef
				}(),
				nil)
			require.ErrorIs(err, writeError)
		})
	})
}

type mockResources struct {
	mock.Mock
}

func (r *mockResources) QueryResource(resource appdef.QName) istructs.IResource {
	return r.Called(resource).Get(0).(istructs.IResource)
}

func (r *mockResources) QueryFunctionArgsBuilder(query istructs.IQueryFunction) istructs.IObjectBuilder {
	return r.Called(query).Get(0).(istructs.IObjectBuilder)
}

func (r *mockResources) Resources(cb func(appdef.QName)) {
	r.Called(cb)
}
