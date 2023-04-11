/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qnames

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/untillpro/voedger/pkg/istorage"
	"github.com/untillpro/voedger/pkg/istorageimpl"
	"github.com/untillpro/voedger/pkg/istructs"
	"github.com/untillpro/voedger/pkg/istructsmem/internal/utils"
	"github.com/untillpro/voedger/pkg/istructsmem/internal/vers"
	"github.com/untillpro/voedger/pkg/schemas"
)

func TestQNames(t *testing.T) {
	sp := istorageimpl.Provide(istorage.ProvideMem())
	storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

	versions := vers.NewVersions()
	if err := versions.Prepare(storage); err != nil {
		panic(err)
	}

	schemaName := istructs.NewQName("test", "schema")
	s := schemas.NewSchemaCache()
	s.Add(schemaName, istructs.SchemaKind_CDoc)
	if err := s.ValidateSchemas(); err != nil {
		panic(err)
	}

	resourceName := istructs.NewQName("test", "resource")
	r := mockResources{}
	r.On("Resources", mock.AnythingOfType("func(istructs.QName)")).
		Run(func(args mock.Arguments) {
			cb := args.Get(0).(func(istructs.QName))
			cb(resourceName)
		})

	names := NewQNames()
	if err := names.Prepare(storage, versions, s, &r); err != nil {
		panic(err)
	}

	require := require.New(t)
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
	})

	t.Run("must be error if unknown name", func(t *testing.T) {
		id, err := names.GetID(istructs.NewQName("test", "unknown"))
		require.Equal(NullQNameID, id)
		require.ErrorIs(err, ErrNameNotFound)
	})

	t.Run("must be error if unknown id", func(t *testing.T) {
		n, err := names.GetQName(QNameID(MaxAvailableQNameID))
		require.Equal(istructs.NullQName, n)
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
		storage.Put(utils.ToBytes(vers.SysQNamesVersion, ver01), []byte(badName), utils.ToBytes(QNameID(512)))

		names := NewQNames()
		err := names.Prepare(storage, versions, nil, nil)
		require.ErrorIs(err, istructs.ErrInvalidQNameStringRepresentation)
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
		storage.Put(utils.ToBytes(vers.SysQNamesVersion, ver01), []byte("test.deleted"), utils.ToBytes(NullQNameID))

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
		storage.Put(utils.ToBytes(vers.SysQNamesVersion, ver01), []byte(istructs.QNameForError.String()), utils.ToBytes(QNameIDForError))

		names := NewQNames()
		err := names.Prepare(storage, versions, nil, nil)
		require.ErrorIs(err, ErrWrongQNameID)
		require.ErrorContains(err, fmt.Sprintf("unexpected ID (%v)", QNameIDForError))
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
