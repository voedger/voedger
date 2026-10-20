/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package cluster

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestValidateAppDeploymentIsReadOnly(t *testing.T) {
	lookupErr := errors.New("descriptor lookup unavailable")
	for _, test := range []struct {
		name       string
		absent     bool
		partitions istructs.NumAppPartitions
		workspaces istructs.NumAppWorkspaces
		viewErr    error
		recordErr  error
		wantErr    error
		message    string
	}{
		{name: "compatible", partitions: 3, workspaces: 4},
		{name: "absent", absent: true},
		{name: "partition count changed", partitions: 2, workspaces: 4, wantErr: ErrNumPartitionsChanged,
			message: "num partitions changed: app test1/app1 declaring NumPartitions=3 but was previously deployed with NumPartitions=2"},
		{name: "workspace count changed", partitions: 3, workspaces: 2, wantErr: ErrNumAppWorkspacesChanged,
			message: "num application workspaces changed: app test1/app1 declaring NumAppWorkspaces=4 but was previously deployed with NumAppWorkspaces=2"},
		{name: "unique lookup failed", viewErr: lookupErr, wantErr: lookupErr},
		{name: "record lookup failed", recordErr: lookupErr, wantErr: lookupErr},
	} {
		t.Run(test.name, func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage(ClusterPackage, ClusterPackageFQN)
			ws := adb.AddWorkspace(appdef.NewQName(ClusterPackage, "workspace"))
			descriptor := appdef.NewQName(ClusterPackage, "descriptor")
			ws.AddCDoc(descriptor)
			ws.SetDescriptor(descriptor)
			app := ws.AddWDoc(qNameWDocApp)
			app.AddField(Field_AppQName, appdef.DataKind_string, true)
			app.AddField(Field_NumPartitions, appdef.DataKind_int32, true)
			app.AddField(Field_NumAppWorkspaces, appdef.DataKind_int32, true)
			app.AddUnique(appdef.NewQName(ClusterPackage, "uniqueAppQName"), []appdef.FieldName{Field_AppQName})
			views := &deploymentDescriptorViews{t: t, err: test.viewErr, absent: test.absent}
			records := &deploymentDescriptorRecords{t: t, err: test.recordErr,
				record: &deploymentDescriptorRecord{partitions: test.partitions, workspaces: test.workspaces}}
			// Only read methods are implemented. Any event access, record apply,
			// view write, or workspace provisioning would panic through the nil interfaces.
			structs := &deploymentDescriptorStructs{def: adb.MustBuild(), views: views, records: records}

			exists, err := ValidateAppDeployment(structs, 1, istructs.AppQName_test1_app1, 3, 4)

			if test.wantErr != nil {
				require.Error(t, err)
				if test.message == "" {
					require.ErrorIs(t, err, test.wantErr)
				} else {
					var responseError coreutils.SysError
					require.ErrorAs(t, err, &responseError)
					require.Equal(t, http.StatusConflict, responseError.HTTPStatus)
					require.Equal(t, test.message, responseError.Message)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, !test.absent, exists)
			}
			require.Equal(t, 1, views.reads)
			if test.absent || test.viewErr != nil {
				require.Zero(t, records.reads)
			} else {
				require.Equal(t, 1, records.reads)
			}
		})
	}
}

type deploymentDescriptorStructs struct {
	istructs.IAppStructs
	def     appdef.IAppDef
	views   istructs.IViewRecords
	records istructs.IRecords
}

func (s *deploymentDescriptorStructs) AppDef() appdef.IAppDef             { return s.def }
func (s *deploymentDescriptorStructs) ViewRecords() istructs.IViewRecords { return s.views }
func (s *deploymentDescriptorStructs) Records() istructs.IRecords         { return s.records }

type deploymentDescriptorViews struct {
	istructs.IViewRecords
	t      *testing.T
	err    error
	absent bool
	reads  int
}

func (v *deploymentDescriptorViews) KeyBuilder(name appdef.QName) istructs.IKeyBuilder {
	require.Equal(v.t, appdef.NewQName(appdef.SysPackage, "Uniques"), name)
	return &deploymentDescriptorKey{}
}

func (v *deploymentDescriptorViews) Get(wsid istructs.WSID, key istructs.IKeyBuilder) (istructs.IValue, error) {
	v.reads++
	require.Equal(v.t, istructs.WSID(1), wsid)
	lookupKey := key.(*deploymentDescriptorKey)
	require.Equal(v.t, []byte(istructs.AppQName_test1_app1.String()), lookupKey.values)
	if v.err != nil {
		return nil, v.err
	}
	if v.absent {
		return nil, istructs.ErrRecordNotFound
	}
	return &deploymentDescriptorValue{}, nil
}

type deploymentDescriptorKey struct {
	istructs.IKeyBuilder
	values []byte
}

func (*deploymentDescriptorKey) PutQName(appdef.FieldName, appdef.QName)      {}
func (*deploymentDescriptorKey) PutInt64(appdef.FieldName, int64)             {}
func (k *deploymentDescriptorKey) PutBytes(_ appdef.FieldName, values []byte) { k.values = values }

type deploymentDescriptorValue struct{ istructs.IValue }

func (*deploymentDescriptorValue) AsRecordID(appdef.FieldName) istructs.RecordID { return 42 }

type deploymentDescriptorRecords struct {
	istructs.IRecords
	t      *testing.T
	err    error
	record istructs.IRecord
	reads  int
}

func (r *deploymentDescriptorRecords) Get(wsid istructs.WSID, _ bool, id istructs.RecordID) (istructs.IRecord, error) {
	r.reads++
	require.Equal(r.t, istructs.WSID(1), wsid)
	require.Equal(r.t, istructs.RecordID(42), id)
	return r.record, r.err
}

type deploymentDescriptorRecord struct {
	istructs.IRecord
	partitions istructs.NumAppPartitions
	workspaces istructs.NumAppWorkspaces
}

func (*deploymentDescriptorRecord) QName() appdef.QName   { return qNameWDocApp }
func (*deploymentDescriptorRecord) ID() istructs.RecordID { return 42 }
func (r *deploymentDescriptorRecord) AsInt32(name appdef.FieldName) int32 {
	if name == Field_NumPartitions {
		return int32(r.partitions)
	}
	if name == Field_NumAppWorkspaces {
		return int32(r.workspaces)
	}
	panic("unexpected descriptor field")
}
