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
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
	"github.com/voedger/voedger/pkg/schemas"
)

func TestQNames(t *testing.T) {
	require := require.New(t)

	sp := istorageimpl.Provide(istorage.ProvideMem())
	storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

	versions := vers.NewVersions()
	if err := versions.Prepare(storage); err != nil {
		panic(err)
	}

	schemaName := schemas.NewQName("test", "schema")

	resourceName := schemas.NewQName("test", "resource")
	r := mockResources{}
	r.On("Resources", mock.AnythingOfType("func(schemas.QName)")).
		Run(func(args mock.Arguments) {
			cb := args.Get(0).(func(schemas.QName))
			cb(resourceName)
		})

	names := NewQNames()
	if err := names.Prepare(storage, versions,
		func() schemas.SchemaCache {
			bld := schemas.NewSchemaCache()
			bld.Add(schemaName, schemas.SchemaKind_CDoc)
			schemas, err := bld.Build()
			require.NoError(err)
			return schemas
		}(),
		&r); err != nil {
		panic(err)
	}

	t.Run("basic QNames methods", func(t *testing.T) {

		check := func(names *QNames, name schemas.QName) QNameID {
			id, err := names.GetID(name)
			require.NoError(err)
			require.NotEqual(NullQNameID, id)

			n, err := names.GetQName(id)
			require.NoError(err)
			require.Equal(name, n)

			return id
		}

		sID := check(names, schemaName)
		rID := check(names, resourceName)

		t.Run("must be ok to load early stored names", func(t *testing.T) {
			versions1 := vers.NewVersions()
			if err := versions1.Prepare(storage); err != nil {
				panic(err)
			}

			names1 := NewQNames()
			if err := names1.Prepare(storage, versions, nil, nil); err != nil {
				panic(err)
			}

			require.Equal(sID, check(names1, schemaName))
			require.Equal(rID, check(names1, resourceName))
		})

		t.Run("must be ok to redeclare names", func(t *testing.T) {
			versions2 := vers.NewVersions()
			if err := versions2.Prepare(storage); err != nil {
				panic(err)
			}

			names2 := NewQNames()
			if err := names2.Prepare(storage, versions,
				func() schemas.SchemaCache {
					bld := schemas.NewSchemaCache()
					bld.Add(schemaName, schemas.SchemaKind_CDoc)
					schemas, err := bld.Build()
					require.NoError(err)
					return schemas
				}(),
				nil); err != nil {
				panic(err)
			}

			require.Equal(sID, check(names2, schemaName))
			require.Equal(rID, check(names2, resourceName))
		})
	})

	t.Run("must be error if unknown name", func(t *testing.T) {
		id, err := names.GetID(schemas.NewQName("test", "unknown"))
		require.Equal(NullQNameID, id)
		require.ErrorIs(err, ErrNameNotFound)
	})

	t.Run("must be error if unknown id", func(t *testing.T) {
		n, err := names.GetQName(QNameID(MaxAvailableQNameID))
		require.Equal(schemas.NullQName, n)
		require.ErrorIs(err, ErrIDNotFound)
	})
}

func TestQNamesPrepareErrors(t *testing.T) {
	require := require.New(t)

	t.Run("must be error if unknown system view version", func(t *testing.T) {
		sp := istorageimpl.Provide(istorage.ProvideMem())
		storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

		versions := vers.NewVersions()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		versions.PutVersion(vers.SysQNamesVersion, lastestVersion+1)

		names := NewQNames()
		err := names.Prepare(storage, versions, nil, nil)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})

	t.Run("must be error if invalid QName readed from system view ", func(t *testing.T) {
		sp := istorageimpl.Provide(istorage.ProvideMem())
		storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

		versions := vers.NewVersions()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		versions.PutVersion(vers.SysQNamesVersion, lastestVersion)
		const badName = "-test.error.qname-"
		storage.Put(utils.ToBytes(consts.SysView_QNames, ver01), []byte(badName), utils.ToBytes(QNameID(512)))

		names := NewQNames()
		err := names.Prepare(storage, versions, nil, nil)
		require.ErrorIs(err, schemas.ErrInvalidQNameStringRepresentation)
		require.ErrorContains(err, badName)
	})

	t.Run("must be ok if deleted QName readed from system view ", func(t *testing.T) {
		sp := istorageimpl.Provide(istorage.ProvideMem())
		storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

		versions := vers.NewVersions()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		versions.PutVersion(vers.SysQNamesVersion, lastestVersion)
		storage.Put(utils.ToBytes(consts.SysView_QNames, ver01), []byte("test.deleted"), utils.ToBytes(NullQNameID))

		names := NewQNames()
		err := names.Prepare(storage, versions, nil, nil)
		require.NoError(err)
	})

	t.Run("must be error if invalid (small) QNameID readed from system view ", func(t *testing.T) {
		sp := istorageimpl.Provide(istorage.ProvideMem())
		storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

		versions := vers.NewVersions()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		versions.PutVersion(vers.SysQNamesVersion, lastestVersion)
		storage.Put(utils.ToBytes(consts.SysView_QNames, ver01), []byte(istructs.QNameForError.String()), utils.ToBytes(QNameIDForError))

		names := NewQNames()
		err := names.Prepare(storage, versions, nil, nil)
		require.ErrorIs(err, ErrWrongQNameID)
		require.ErrorContains(err, fmt.Sprintf("unexpected ID (%v)", QNameIDForError))
	})

	t.Run("must be error if too many QNames", func(t *testing.T) {
		sp := istorageimpl.Provide(istorage.ProvideMem())
		storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

		versions := vers.NewVersions()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		names := NewQNames()
		err := names.Prepare(storage, versions,
			func() schemas.SchemaCache {
				bld := schemas.NewSchemaCache()
				for i := 0; i <= MaxAvailableQNameID; i++ {
					bld.Add(schemas.NewQName("test", fmt.Sprintf("name_%d", i)), schemas.SchemaKind_Object)
				}
				schemas, err := bld.Build()
				require.NoError(err)
				return schemas
			}(),
			nil)
		require.ErrorIs(err, ErrQNameIDsExceeds)
	})

	t.Run("must be error if write to storage failed", func(t *testing.T) {
		qName := schemas.NewQName("test", "test")
		writeError := errors.New("storage write error")

		t.Run("must be error if write some name failed", func(t *testing.T) {
			storage := teststore.NewTestStorage()

			versions := vers.NewVersions()
			if err := versions.Prepare(storage); err != nil {
				panic(err)
			}

			storage.SchedulePutError(writeError, utils.ToBytes(consts.SysView_QNames, ver01), []byte(qName.String()))

			names := NewQNames()
			err := names.Prepare(storage, versions,
				func() schemas.SchemaCache {
					bld := schemas.NewSchemaCache()
					bld.Add(qName, schemas.SchemaKind_Object)
					schemas, err := bld.Build()
					require.NoError(err)
					return schemas
				}(),
				nil)
			require.ErrorIs(err, writeError)
		})

		t.Run("must be error if write system view version failed", func(t *testing.T) {
			storage := teststore.NewTestStorage()

			versions := vers.NewVersions()
			if err := versions.Prepare(storage); err != nil {
				panic(err)
			}

			storage.SchedulePutError(writeError, utils.ToBytes(consts.SysView_Versions), utils.ToBytes(vers.SysQNamesVersion))

			names := NewQNames()
			err := names.Prepare(storage, versions,
				func() schemas.SchemaCache {
					bld := schemas.NewSchemaCache()
					bld.Add(qName, schemas.SchemaKind_Object)
					schemas, err := bld.Build()
					require.NoError(err)
					return schemas
				}(),
				nil)
			require.ErrorIs(err, writeError)
		})
	})
}

type mockResources struct {
	mock.Mock
}

func (r *mockResources) QueryResource(resource schemas.QName) istructs.IResource {
	return r.Called(resource).Get(0).(istructs.IResource)
}

func (r *mockResources) QueryFunctionArgsBuilder(query istructs.IQueryFunction) istructs.IObjectBuilder {
	return r.Called(query).Get(0).(istructs.IObjectBuilder)
}

func (r *mockResources) Resources(cb func(schemas.QName)) {
	r.Called(cb)
}
